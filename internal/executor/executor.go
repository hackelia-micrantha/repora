// Package executor owns reconciliation side effects.
//
// It executes planner-produced actions without inspecting repository status or
// re-deciding reconciliation policy. Callers must provide an already-built,
// validated in-memory plan.
package executor

import (
	"fmt"
	"strings"

	"repoctl/internal/plan"
)

// Git contains only the mutation operations required to execute plans.
type Git interface {
	PushBranch(repoPath, remote, srcRef, dstBranch string) error
	ForcePushBranchWithLease(repoPath, remote, srcRef, dstBranch, expectedOldOID string) error
}

// ActionResult records whether one planned action was applied. This is an
// internal execution result, not a stabilized public serialization contract.
type ActionResult struct {
	Action  plan.PlannedAction
	Applied bool
}

// Result contains action results in the same deterministic order as the plan.
type Result struct {
	Actions []ActionResult
}

// Execute applies every action in planned order. It validates the complete plan
// before invoking Git so malformed plans fail closed without mutation.
func Execute(repoPath string, planned plan.ReconciliationPlan, git Git) (Result, error) {
	result := Result{Actions: make([]ActionResult, 0, len(planned.Actions))}
	for i, action := range planned.Actions {
		if err := validateAction(action); err != nil {
			return result, fmt.Errorf("validate action %d: %w", i, err)
		}
	}

	for _, action := range planned.Actions {
		actionResult := ActionResult{Action: action}
		srcRef := remoteTrackingRef(action.Source.Name, action.Source.Branch)
		if action.Force {
			if err := git.ForcePushBranchWithLease(repoPath, action.Target.Name, srcRef, action.Target.Branch, action.ExpectedOldTarget); err != nil {
				return result, fmt.Errorf("force push branch: %w", err)
			}
		} else {
			if err := git.PushBranch(repoPath, action.Target.Name, srcRef, action.Target.Branch); err != nil {
				return result, fmt.Errorf("push branch: %w", err)
			}
		}
		actionResult.Applied = true
		result.Actions = append(result.Actions, actionResult)
	}
	return result, nil
}

func validateAction(action plan.PlannedAction) error {
	if action.Type != plan.ActionPushBranch {
		return fmt.Errorf("unsupported action type %q", action.Type)
	}
	if strings.TrimSpace(action.Source.Name) == "" || strings.TrimSpace(action.Source.Branch) == "" {
		return fmt.Errorf("push action requires source remote and branch")
	}
	if strings.TrimSpace(action.Target.Name) == "" || strings.TrimSpace(action.Target.Branch) == "" {
		return fmt.Errorf("push action requires target remote and branch")
	}
	if action.Force && strings.TrimSpace(action.ExpectedOldTarget) == "" {
		return fmt.Errorf("forced push action requires expected old target")
	}
	if !action.Force && strings.TrimSpace(action.ExpectedOldTarget) != "" {
		return fmt.Errorf("normal push action must not include expected old target")
	}
	return nil
}

func remoteTrackingRef(remote, branch string) string {
	return "refs/remotes/" + strings.TrimSpace(remote) + "/" + strings.TrimSpace(branch)
}
