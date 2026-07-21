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

type Outcome string

const (
	OutcomeApplied Outcome = "APPLIED"
	OutcomeFailed  Outcome = "FAILED"
	OutcomeSkipped Outcome = "SKIPPED"
)

// ActionResult records the outcome of one planned action. This is an internal
// execution result, not a stabilized public serialization contract.
type ActionResult struct {
	Index   int
	Action  plan.PlannedAction
	Outcome Outcome
	Error   string
}

// Result contains one action result for every planned action in deterministic
// plan order, including actions skipped after a failure.
type Result struct {
	Actions []ActionResult
}

func (r Result) AllApplied() bool {
	if len(r.Actions) == 0 {
		return false
	}
	for _, action := range r.Actions {
		if action.Outcome != OutcomeApplied {
			return false
		}
	}
	return true
}

// Execute applies every action in planned order. It validates the complete plan
// before invoking Git so malformed plans fail closed without mutation. Mutation
// failure stops execution while preserving applied, failed, and skipped results.
func Execute(repoPath string, planned plan.ReconciliationPlan, git Git) (Result, error) {
	result := Result{Actions: make([]ActionResult, len(planned.Actions))}
	for i, action := range planned.Actions {
		result.Actions[i] = ActionResult{Index: i, Action: action, Outcome: OutcomeSkipped}
	}

	for i, action := range planned.Actions {
		if err := validateAction(action); err != nil {
			result.Actions[i].Outcome = OutcomeFailed
			result.Actions[i].Error = err.Error()
			return result, fmt.Errorf("validate action %d: %w", i, err)
		}
	}

	for i, action := range planned.Actions {
		srcRef := remoteTrackingRef(action.Source.Name, action.Source.Branch)
		var err error
		if action.Force {
			err = git.ForcePushBranchWithLease(repoPath, action.Target.Name, srcRef, action.Target.Branch, action.ExpectedOldTarget)
		} else {
			err = git.PushBranch(repoPath, action.Target.Name, srcRef, action.Target.Branch)
		}
		if err != nil {
			result.Actions[i].Outcome = OutcomeFailed
			result.Actions[i].Error = err.Error()
			return result, fmt.Errorf("execute action %d: %w", i, err)
		}
		result.Actions[i].Outcome = OutcomeApplied
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
