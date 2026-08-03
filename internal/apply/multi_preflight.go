package apply

import (
	"errors"
	"fmt"
	"strings"

	"repoctl/internal/config"
	"repoctl/internal/executor"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/journal"
	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
	"repoctl/internal/refpolicy"
	"repoctl/internal/status"
)

// PreflightRepositoryArtifactAudited validates one exact path-bound artifact
// against current topology and observations, writes one repository-level intent
// and result pair, and performs no remote mutation.
func PreflightRepositoryArtifactAudited(repo config.Repo, observed status.RepositoryResult, artifact planartifact.Artifact, git Git, audit Audit) (Result, error) {
	result := Result{
		ID:      repo.ID,
		UID:     repo.DurableID(),
		DryRun:  true,
		Actions: []Action{},
	}
	if strings.TrimSpace(audit.ExecutionID) == "" {
		return result, fmt.Errorf("audit execution ID is required")
	}
	if audit.Writer == nil {
		return result, fmt.Errorf("audit journal writer is required")
	}

	planned, bindings, aggregateState, err := validateRepositoryArtifact(repo, observed, artifact)
	result.State = aggregateState
	if err != nil {
		return result, err
	}
	result.Actions = stableActions(planned)

	path := ""
	if len(planned.Actions) > 0 {
		path, err = gitwrap.MirrorPath(repo.DurableID())
		if err != nil {
			return result, err
		}
		if err := validateBoundDefaultBranches(path, planned, bindings, git); err != nil {
			return result, fmt.Errorf("validate plan scope for repo %q: %w", repo.ID, err)
		}
	}

	intent, err := journal.FromPlan(audit.ExecutionID, journal.ModeDryRun, artifact)
	if err != nil {
		return result, fmt.Errorf("create journal intent for repo %q: %w", repo.ID, err)
	}
	intentRef, err := audit.Writer.Write(intent)
	result.Journal = &JournalReferences{ExecutionID: audit.ExecutionID, Intent: intentRef}
	if err != nil {
		return result, fmt.Errorf("persist journal intent for repo %q: %w", repo.ID, err)
	}

	preflight := executor.Result{Actions: []executor.ActionResult{}}
	var preflightErr error
	if len(planned.Actions) > 0 {
		preflight, preflightErr = executor.PreflightWithBindings(path, artifact, git, bindings)
	}
	journalErr := persistPreflightResult(&result, &audit, artifact, preflight, preflightErr)
	if joined := errors.Join(preflightErr, journalErr); joined != nil {
		return result, fmt.Errorf("dry-run plan artifact for repo %q: %w", repo.ID, joined)
	}
	return result, nil
}

func validateRepositoryArtifact(repo config.Repo, observed status.RepositoryResult, artifact planartifact.Artifact) (plan.ReconciliationPlan, executor.RuntimeBindings, status.State, error) {
	empty := plan.ReconciliationPlan{}
	bindings := executor.RuntimeBindings{SourceRemote: "canonical", TargetRemotes: map[string]string{}}
	if artifact.Version != planartifact.Version {
		return empty, bindings, "", fmt.Errorf("path-bound execution requires reconciliation artifact version %d", planartifact.Version)
	}
	plans, err := artifact.Plans()
	if err != nil {
		return empty, bindings, "", fmt.Errorf("validate plan artifact: %w", err)
	}
	if len(plans) != 1 {
		return empty, bindings, "", fmt.Errorf("plan artifact requires exactly one repository, got %d", len(plans))
	}
	planned := plans[0]
	if planned.UID != repo.DurableID() {
		return empty, bindings, "", fmt.Errorf("plan repository uid %q does not match configured uid %q", planned.UID, repo.DurableID())
	}
	if len(repo.Mirrors) == 0 {
		return empty, bindings, "", fmt.Errorf("repo %q requires at least one configured mirror", repo.ID)
	}
	if observed.ID != "" && observed.ID != repo.ID {
		return empty, bindings, "", fmt.Errorf("status repository id %q does not match configured id %q", observed.ID, repo.ID)
	}
	if observed.UID != "" && observed.UID != repo.DurableID() {
		return empty, bindings, "", fmt.Errorf("status repository uid %q does not match configured uid %q", observed.UID, repo.DurableID())
	}
	if strings.TrimSpace(observed.Error) != "" {
		return empty, bindings, "", fmt.Errorf("repo %q status is incomplete: %s", repo.ID, observed.Error)
	}
	if len(observed.Mirrors) != len(repo.Mirrors) {
		return empty, bindings, "", fmt.Errorf("repo %q status contains %d mirrors, want %d", repo.ID, len(observed.Mirrors), len(repo.Mirrors))
	}

	canonicalPath, err := repo.Canonical.RepositoryPath()
	if err != nil {
		return empty, bindings, "", fmt.Errorf("resolve configured canonical identity: %w", err)
	}
	policy, err := repo.EffectiveRefPolicy()
	if err != nil {
		return empty, bindings, "", fmt.Errorf("invalid ref policy for repo %q: %w", repo.ID, err)
	}

	observedByTarget := make(map[string]status.MirrorResult, len(observed.Mirrors))
	for i, mirror := range observed.Mirrors {
		target := strings.TrimSpace(mirror.Target)
		if target == "" {
			return empty, bindings, "", fmt.Errorf("repo %q status mirror %d has no stable target identity", repo.ID, i)
		}
		if _, exists := observedByTarget[target]; exists {
			return empty, bindings, "", fmt.Errorf("repo %q status duplicates mirror target %q", repo.ID, target)
		}
		observedByTarget[target] = mirror
	}

	actionsByTarget := make(map[string]plan.PlannedAction, len(planned.Actions))
	for i, action := range planned.Actions {
		if action.Source.Provider != repo.Canonical.Provider || action.Source.Path != canonicalPath {
			return empty, bindings, "", fmt.Errorf("plan action %d source does not match configured canonical repository", i)
		}
		target := stableRemoteID(action.Target)
		if _, exists := actionsByTarget[target]; exists {
			return empty, bindings, "", fmt.Errorf("plan duplicates mirror target %q", target)
		}
		actionsByTarget[target] = action
	}

	aggregate := status.StateEqual
	configuredTargets := make(map[string]struct{}, len(repo.Mirrors))
	for i, endpoint := range repo.Mirrors {
		target, err := endpoint.TargetID()
		if err != nil {
			return empty, bindings, "", fmt.Errorf("resolve configured mirror %d identity: %w", i, err)
		}
		configuredTargets[target] = struct{}{}
		bindings.TargetRemotes[target] = mirrorRemoteName(len(repo.Mirrors), i)

		mirror, ok := observedByTarget[target]
		if !ok {
			return empty, bindings, "", fmt.Errorf("repo %q status is missing configured mirror %q", repo.ID, target)
		}
		if mirror.State == status.StateError || strings.TrimSpace(mirror.Error) != "" {
			return empty, bindings, "", fmt.Errorf("repo %q mirror %q status is incomplete: %s", repo.ID, target, mirror.Error)
		}
		decision, err := policy.Decide(refpolicy.Relationship(mirror.State))
		if err != nil {
			return empty, bindings, "", fmt.Errorf("repo %q mirror %q has unsupported state %q: %w", repo.ID, target, mirror.State, err)
		}
		action, hasAction := actionsByTarget[target]
		if decision.Action != hasAction {
			return empty, bindings, "", fmt.Errorf("repo %q plan is stale or policy-invalid for mirror %q state %s", repo.ID, target, mirror.State)
		}
		if hasAction {
			if action.Force != decision.Force {
				return empty, bindings, "", fmt.Errorf("repo %q plan force intent does not match mirror %q state %s", repo.ID, target, mirror.State)
			}
			path, err := endpoint.RepositoryPath()
			if err != nil {
				return empty, bindings, "", fmt.Errorf("resolve configured mirror %q path: %w", target, err)
			}
			if action.Target.Provider != endpoint.Provider || action.Target.Path != path {
				return empty, bindings, "", fmt.Errorf("plan target %q does not match configured mirror %q", stableRemoteID(action.Target), target)
			}
		}
		aggregate = combineState(aggregate, mirror.State)
	}
	for target := range actionsByTarget {
		if _, ok := configuredTargets[target]; !ok {
			return empty, bindings, "", fmt.Errorf("plan targets unknown configured mirror %q", target)
		}
	}
	return planned, bindings, aggregate, nil
}

func validateBoundDefaultBranches(repoPath string, planned plan.ReconciliationPlan, bindings executor.RuntimeBindings, git Git) error {
	canonicalBranch, err := git.ResolveRemoteHeadBranch(repoPath, bindings.SourceRemote)
	if err != nil {
		return fmt.Errorf("resolve current canonical HEAD: %w", err)
	}
	canonicalBranch = strings.TrimSpace(canonicalBranch)
	if canonicalBranch == "" {
		return fmt.Errorf("current canonical default branch is empty")
	}
	for i, action := range planned.Actions {
		target := stableRemoteID(action.Target)
		remote, ok := bindings.TargetRemotes[target]
		if !ok {
			return fmt.Errorf("action %d has no runtime binding for target %s", i, target)
		}
		mirrorBranch, err := git.ResolveRemoteHeadBranch(repoPath, remote)
		if err != nil {
			return fmt.Errorf("resolve current mirror %s HEAD: %w", target, err)
		}
		mirrorBranch = strings.TrimSpace(mirrorBranch)
		if mirrorBranch == "" {
			mirrorBranch = canonicalBranch
		}
		if action.Source.Branch != canonicalBranch || action.Target.Branch != mirrorBranch {
			return fmt.Errorf("action %d targets %s/%s but current default branches are %s/%s", i, action.Source.Branch, action.Target.Branch, canonicalBranch, mirrorBranch)
		}
	}
	return nil
}

func stableActions(planned plan.ReconciliationPlan) []Action {
	actions := make([]Action, 0, len(planned.Actions))
	for _, plannedAction := range planned.Actions {
		action := Action{
			Type:   string(plannedAction.Type),
			Source: stableRemoteID(plannedAction.Source) + "/" + plannedAction.Source.Branch,
			Target: stableRemoteID(plannedAction.Target) + "/" + plannedAction.Target.Branch,
			Force:  plannedAction.Force,
		}
		if plannedAction.Force {
			action.ExpectedOldTarget = plannedAction.ExpectedOldTarget
		}
		actions = append(actions, action)
	}
	return actions
}

func stableRemoteID(remote plan.Remote) string {
	return strings.TrimSpace(remote.Provider) + ":" + strings.Trim(strings.TrimSpace(remote.Path), "/")
}

func combineState(current, next status.State) status.State {
	if next == status.StateDiverged || current == status.StateDiverged {
		return status.StateDiverged
	}
	if next == status.StateAhead || current == status.StateAhead {
		return status.StateAhead
	}
	if next == status.StateBehind || current == status.StateBehind {
		return status.StateBehind
	}
	return status.StateEqual
}
