package apply

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"repoctl/internal/config"
	"repoctl/internal/executor"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/journal"
	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

var ErrForceAuthorization = errors.New("destructive plan requires --force")
var publicEmbeddedAbsolutePathPattern = regexp.MustCompile(`(^|[[:space:]'"])/[^[:space:]'"]+`)

// ExecuteRepositoryArtifactAudited applies or dry-runs one exact path-bound
// repository artifact. It shares topology, policy, branch, intent, and OID
// preflight across both modes. Real execution attempts every independent mirror
// action in deterministic artifact order and does not roll back earlier success.
func ExecuteRepositoryArtifactAudited(repo config.Repo, observed status.RepositoryResult, artifact planartifact.Artifact, git Git, allowForce, dryRun bool, audit Audit) (DetailedResult, error) {
	result := DetailedResult{
		ID:      repo.ID,
		UID:     repo.DurableID(),
		DryRun:  dryRun,
		Actions: []DetailedAction{},
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
	result.Actions = plannedDetailedActions(planned)
	if !dryRun && planRequiresForce(planned) && !allowForce {
		return result, fmt.Errorf("repo %q: %w", repo.ID, ErrForceAuthorization)
	}

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

	mode := journal.ModeApply
	if dryRun {
		mode = journal.ModeDryRun
	}
	intent, err := journal.FromPlan(audit.ExecutionID, mode, artifact)
	if err != nil {
		return result, fmt.Errorf("create journal intent for repo %q: %w", repo.ID, err)
	}
	intentRef, err := audit.Writer.Write(intent)
	result.Journal = &JournalReferences{ExecutionID: audit.ExecutionID, Intent: intentRef}
	if err != nil {
		return result, fmt.Errorf("persist journal intent for repo %q: %w", repo.ID, err)
	}

	executed := executor.Result{Actions: []executor.ActionResult{}}
	var executionErr error
	if len(planned.Actions) > 0 {
		if dryRun {
			executed, executionErr = executor.PreflightWithBindings(path, artifact, git, bindings)
		} else {
			executed, executionErr = executor.ExecuteWithBindings(path, artifact, git, bindings)
		}
	}
	result.Actions = detailedActions(planned, executed, dryRun, executionErr)
	result.Applied = !dryRun && (len(planned.Actions) == 0 || executed.AllApplied())

	compat := Result{Journal: result.Journal}
	var journalErr error
	if dryRun {
		journalErr = persistPreflightResult(&compat, &audit, artifact, executed, executionErr)
	} else {
		journalErr = persistExecutionResult(&compat, &audit, artifact, executed)
	}
	result.Journal = compat.Journal

	if joined := errors.Join(executionErr, journalErr); joined != nil {
		diagnostic := publicDiagnostic(joined.Error())
		result.Error = diagnostic
		return result, fmt.Errorf("execute plan artifact for repo %q: %s", repo.ID, diagnostic)
	}
	return result, nil
}

func plannedDetailedActions(planned plan.ReconciliationPlan) []DetailedAction {
	actions := make([]DetailedAction, 0, len(planned.Actions))
	for _, action := range planned.Actions {
		actions = append(actions, DetailedAction{
			Type:    string(action.Type),
			Source:  stableRemoteID(action.Source) + "/" + action.Source.Branch,
			Target:  stableRemoteID(action.Target) + "/" + action.Target.Branch,
			Force:   action.Force,
			Before:  action.ExpectedOldTarget,
			Desired: action.ExpectedSource,
			Outcome: string(journal.OutcomePlanned),
		})
	}
	return actions
}

func detailedActions(planned plan.ReconciliationPlan, executed executor.Result, dryRun bool, executionErr error) []DetailedAction {
	actions := plannedDetailedActions(planned)
	if len(executed.Actions) != len(actions) {
		return actions
	}
	for i, evidence := range executed.Actions {
		action := &actions[i]
		switch evidence.Outcome {
		case executor.OutcomeApplied:
			action.Outcome = string(journal.OutcomeApplied)
			action.After = strings.TrimSpace(evidence.AfterOID)
		case executor.OutcomeFailed:
			if evidence.Stale {
				action.Outcome = string(journal.OutcomeStale)
			} else {
				action.Outcome = string(journal.OutcomeFailed)
			}
			action.Error = publicDiagnostic(evidence.Error)
		case executor.OutcomeSkipped:
			if dryRun && executionErr == nil {
				action.Outcome = string(journal.OutcomeValidated)
			} else {
				action.Outcome = string(journal.OutcomeSkipped)
			}
		default:
			action.Outcome = string(journal.OutcomeFailed)
			action.Error = "unsupported executor outcome"
		}
	}
	return actions
}

func publicDiagnostic(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "execution failed"
	}
	value = publicEmbeddedAbsolutePathPattern.ReplaceAllString(value, `${1}[REDACTED_PATH]`)
	lower := strings.ToLower(value)
	if strings.Contains(lower, "://") || strings.Contains(lower, "token=") || strings.Contains(lower, "password=") || strings.Contains(lower, "authorization:") || strings.Contains(lower, "\\") || strings.HasPrefix(lower, "/") || strings.HasPrefix(lower, "file:") || strings.Contains(lower, "@") {
		return "execution diagnostic redacted"
	}
	return value
}
