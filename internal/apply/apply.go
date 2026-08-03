package apply

import (
	"fmt"
	"strings"

	"repoctl/internal/config"
	"repoctl/internal/executor"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

const (
	OutputKind    = "repora.apply"
	OutputVersion = 2
)

type Output struct {
	Kind    string   `json:"kind"`
	Version int      `json:"version"`
	Results []Result `json:"results"`
}

type Result struct {
	ID      string             `json:"id"`
	UID     string             `json:"uid"`
	State   status.State       `json:"state"`
	Applied bool               `json:"applied"`
	DryRun  bool               `json:"dry_run"`
	Actions []Action           `json:"actions"`
	Journal *JournalReferences `json:"journal,omitempty"`
	Error   string             `json:"error,omitempty"`
}

type JournalReferences struct {
	ExecutionID string `json:"execution_id"`
	Intent      string `json:"intent"`
	Result      string `json:"result,omitempty"`
}

type Action struct {
	Type              string `json:"type"`
	Source            string `json:"source"`
	Target            string `json:"target"`
	Force             bool   `json:"force"`
	ExpectedOldTarget string `json:"expected_old_target,omitempty"`
}

type Git interface {
	executor.Git
	ResolveRemoteHeadBranch(repoPath, remote string) (string, error)
}

func IsUnsafe(result status.Result) bool {
	return plan.RequiresMirrorHeadObservation(result)
}

func BuildArtifact(repo config.Repo, st status.Result, git Git) (planartifact.Artifact, error) {
	planned, err := buildPlan(repo, st, git)
	if err != nil {
		return planartifact.Artifact{}, err
	}
	artifact, err := planartifact.FromCurrentPlans(planned)
	if err != nil {
		return planartifact.Artifact{}, fmt.Errorf("validate plan artifact for repo %q: %w", repo.ID, err)
	}
	return artifact, nil
}

func Execute(repo config.Repo, st status.Result, git Git, force bool, dryRun bool) (Result, error) {
	artifact, err := BuildArtifact(repo, st, git)
	if err != nil {
		return newResult(repo, st, dryRun), err
	}
	return ExecuteArtifact(repo, st, artifact, git, force, dryRun)
}

func ExecuteArtifact(repo config.Repo, st status.Result, artifact planartifact.Artifact, git Git, allowForce bool, dryRun bool) (Result, error) {
	return executeArtifact(repo, st, artifact, git, allowForce, dryRun, nil)
}

func ArtifactRequiresForce(artifact planartifact.Artifact) (bool, error) {
	plans, err := artifact.Plans()
	if err != nil {
		return false, err
	}
	for _, planned := range plans {
		if planRequiresForce(planned) {
			return true, nil
		}
	}
	return false, nil
}

func newResult(repo config.Repo, st status.Result, dryRun bool) Result {
	return Result{
		ID:      repo.ID,
		UID:     repo.DurableID(),
		State:   st.State,
		DryRun:  dryRun,
		Actions: []Action{},
	}
}

func buildPlan(repo config.Repo, st status.Result, git Git) (plan.ReconciliationPlan, error) {
	path, err := gitwrap.MirrorPath(repo.DurableID())
	if err != nil {
		return plan.ReconciliationPlan{}, err
	}

	srcBranch, err := git.ResolveRemoteHeadBranch(path, "canonical")
	if err != nil {
		return plan.ReconciliationPlan{}, fmt.Errorf("resolve canonical HEAD for repo %q: %w", repo.ID, err)
	}
	dstBranch, err := git.ResolveRemoteHeadBranch(path, "mirror")
	if err != nil {
		return plan.ReconciliationPlan{}, fmt.Errorf("resolve mirror HEAD for repo %q: %w", repo.ID, err)
	}
	if dstBranch == "" {
		dstBranch = srcBranch
	}

	observation := plan.Observation{CanonicalBranch: srcBranch, MirrorBranch: dstBranch}
	if plan.RequiresRefObservation(st) {
		srcRef := "refs/remotes/canonical/" + srcBranch
		sourceOID, err := git.ResolveRevision(path, srcRef)
		if err != nil {
			return plan.ReconciliationPlan{}, fmt.Errorf("resolve canonical branch for repo %q: %w", repo.ID, err)
		}
		dstRef := "refs/remotes/mirror/" + dstBranch
		targetOID, err := git.ResolveRevision(path, dstRef)
		if err != nil {
			return plan.ReconciliationPlan{}, fmt.Errorf("resolve mirror branch for repo %q: %w", repo.ID, err)
		}
		observation.CanonicalHeadOID = sourceOID
		observation.MirrorHeadOID = targetOID
	}

	return plan.Reconcile(repo, st, observation, true)
}

func planForRepository(repo config.Repo, artifact planartifact.Artifact) (plan.ReconciliationPlan, error) {
	plans, err := artifact.Plans()
	if err != nil {
		return plan.ReconciliationPlan{}, fmt.Errorf("validate plan artifact: %w", err)
	}
	if len(plans) != 1 {
		return plan.ReconciliationPlan{}, fmt.Errorf("plan artifact requires exactly one repository, got %d", len(plans))
	}
	planned := plans[0]
	if planned.UID != repo.DurableID() {
		return plan.ReconciliationPlan{}, fmt.Errorf("plan repository uid %q does not match configured uid %q", planned.UID, repo.DurableID())
	}
	if len(repo.Mirrors) != 1 {
		return plan.ReconciliationPlan{}, fmt.Errorf("repo %q requires exactly one configured mirror, got %d", repo.ID, len(repo.Mirrors))
	}
	if len(planned.Actions) > 1 {
		return plan.ReconciliationPlan{}, fmt.Errorf("repo %q plan supports at most one default-branch action, got %d", repo.ID, len(planned.Actions))
	}

	canonicalPath, err := repo.Canonical.RepositoryPath()
	if err != nil {
		return plan.ReconciliationPlan{}, fmt.Errorf("resolve configured canonical identity: %w", err)
	}
	mirrorPath, err := repo.Mirrors[0].RepositoryPath()
	if err != nil {
		return plan.ReconciliationPlan{}, fmt.Errorf("resolve configured mirror identity: %w", err)
	}
	for i, action := range planned.Actions {
		if action.Source.Provider != repo.Canonical.Provider || action.Source.Name != "canonical" {
			return plan.ReconciliationPlan{}, fmt.Errorf("plan action %d source does not match configured canonical repository", i)
		}
		if action.Target.Provider != repo.Mirrors[0].Provider || action.Target.Name != "mirror" {
			return plan.ReconciliationPlan{}, fmt.Errorf("plan action %d target does not match configured mirror repository", i)
		}
		if artifact.Version == planartifact.Version {
			if action.Source.Path != canonicalPath {
				return plan.ReconciliationPlan{}, fmt.Errorf("plan action %d source path %q does not match configured canonical path %q", i, action.Source.Path, canonicalPath)
			}
			if action.Target.Path != mirrorPath {
				return plan.ReconciliationPlan{}, fmt.Errorf("plan action %d target path %q does not match configured mirror path %q", i, action.Target.Path, mirrorPath)
			}
		}
	}
	return planned, nil
}

func validateStateIntent(repoID string, state status.State, planned plan.ReconciliationPlan) error {
	actionCount := len(planned.Actions)
	switch state {
	case status.StateEqual:
		if actionCount != 0 {
			return fmt.Errorf("repo %q plan is stale: current state is EQUAL but artifact contains %d action(s)", repoID, actionCount)
		}
		return nil
	case status.StateBehind:
		if actionCount != 1 || planned.Actions[0].Force {
			return fmt.Errorf("repo %q plan is stale or policy-invalid: BEHIND requires one non-forced action", repoID)
		}
		return nil
	case status.StateAhead, status.StateDiverged:
		if actionCount != 1 || !planned.Actions[0].Force {
			return fmt.Errorf("repo %q plan is stale or policy-invalid: %s requires one forced action", repoID, state)
		}
		return nil
	default:
		return fmt.Errorf("repo %q has unsupported current state %q", repoID, state)
	}
}

func validateDefaultBranchScope(repoPath string, planned plan.ReconciliationPlan, git Git) error {
	canonicalBranch, err := git.ResolveRemoteHeadBranch(repoPath, "canonical")
	if err != nil {
		return fmt.Errorf("resolve current canonical HEAD: %w", err)
	}
	mirrorBranch, err := git.ResolveRemoteHeadBranch(repoPath, "mirror")
	if err != nil {
		return fmt.Errorf("resolve current mirror HEAD: %w", err)
	}
	canonicalBranch = strings.TrimSpace(canonicalBranch)
	mirrorBranch = strings.TrimSpace(mirrorBranch)
	if mirrorBranch == "" {
		mirrorBranch = canonicalBranch
	}
	for i, action := range planned.Actions {
		if action.Source.Branch != canonicalBranch || action.Target.Branch != mirrorBranch {
			return fmt.Errorf("action %d targets %s/%s but current default branches are %s/%s", i, action.Source.Branch, action.Target.Branch, canonicalBranch, mirrorBranch)
		}
	}
	return nil
}

func planRequiresForce(planned plan.ReconciliationPlan) bool {
	for _, action := range planned.Actions {
		if action.Force {
			return true
		}
	}
	return false
}

func compatibilityActions(planned plan.ReconciliationPlan) []Action {
	actions := make([]Action, 0, len(planned.Actions))
	for _, plannedAction := range planned.Actions {
		action := Action{
			Type:   string(plannedAction.Type),
			Source: plannedAction.Source.Name + "/" + plannedAction.Source.Branch,
			Target: plannedAction.Target.Provider + "/" + plannedAction.Target.Branch,
			Force:  plannedAction.Force,
		}
		if plannedAction.Force {
			action.ExpectedOldTarget = plannedAction.ExpectedOldTarget
		}
		actions = append(actions, action)
	}
	return actions
}
