package executor

import (
	"errors"
	"fmt"

	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
)

// ExecuteWithBindings performs complete all-action preflight and then attempts
// every independent action in deterministic artifact order. Runtime failure of
// one target does not prevent later targets from being attempted. Successful
// earlier actions are not rolled back.
func ExecuteWithBindings(repoPath string, artifact planartifact.Artifact, git Git, bindings RuntimeBindings) (Result, error) {
	planned, err := onePlan(artifact)
	if err != nil {
		return Result{}, err
	}
	result, err := preflightPlan(repoPath, planned, git, &bindings)
	if err != nil {
		return result, err
	}

	var joined error
	failed := 0
	for i, action := range planned.Actions {
		sourceRemote := bindings.SourceRemote
		targetRemote := bindings.TargetRemotes[stableTargetID(action.Target)]
		sourceRef := remoteTrackingRef(sourceRemote, action.Source.Branch)

		var actionErr error
		if action.Force {
			actionErr = git.ForcePushBranchWithLease(repoPath, targetRemote, sourceRef, action.Target.Branch, action.ExpectedOldTarget)
		} else {
			actionErr = git.PushBranch(repoPath, targetRemote, sourceRef, action.Target.Branch)
		}
		if actionErr != nil {
			failed++
			result.Actions[i].Outcome = OutcomeFailed
			result.Actions[i].Error = actionErr.Error()
			joined = errors.Join(joined, fmt.Errorf("action %d target %s: %w", i, stableTargetID(action.Target), actionErr))
			continue
		}
		result.Actions[i].Outcome = OutcomeApplied
		result.Actions[i].AfterOID = action.ExpectedSource
	}
	if joined != nil {
		return result, fmt.Errorf("%d mirror action(s) failed: %w", failed, joined)
	}
	return result, nil
}

// Keep plan imported for documentation tooling that follows the concrete
// action type from this execution boundary.
var _ plan.PlannedAction
