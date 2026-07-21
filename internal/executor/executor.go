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

// Git contains the reference reads and mutation operations required to execute
// plans safely. Reference reads are used only to reject stale plan input.
type Git interface {
	ResolveRevision(repoPath, rev string) (string, error)
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
// and verifies every expected source and target reference before invoking a Git
// mutation. Malformed or stale plans therefore fail closed without mutation.
func Execute(repoPath string, planned plan.ReconciliationPlan, git Git) (Result, error) {
	result := Result{Actions: make([]ActionResult, len(planned.Actions))}
	for i, action := range planned.Actions {
		result.Actions[i] = ActionResult{Index: i, Action: action, Outcome: OutcomeSkipped}
	}

	for i, action := range planned.Actions {
		if err := validateAction(action); err != nil {
			return failPreflight(result, i, fmt.Errorf("validate action %d: %w", i, err))
		}
	}
	for i, action := range planned.Actions {
		if err := validateCurrentRefs(repoPath, action, git); err != nil {
			return failPreflight(result, i, fmt.Errorf("stale action %d: %w", i, err))
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

func failPreflight(result Result, index int, err error) (Result, error) {
	result.Actions[index].Outcome = OutcomeFailed
	result.Actions[index].Error = err.Error()
	return result, err
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
	if strings.TrimSpace(action.ExpectedSource) == "" {
		return fmt.Errorf("push action requires expected source")
	}
	if strings.TrimSpace(action.ExpectedOldTarget) == "" {
		return fmt.Errorf("push action requires expected old target")
	}
	return nil
}

func validateCurrentRefs(repoPath string, action plan.PlannedAction, git Git) error {
	sourceRef := remoteTrackingRef(action.Source.Name, action.Source.Branch)
	currentSource, err := git.ResolveRevision(repoPath, sourceRef)
	if err != nil {
		return fmt.Errorf("resolve source %s: %w", sourceRef, err)
	}
	if strings.TrimSpace(currentSource) != strings.TrimSpace(action.ExpectedSource) {
		return fmt.Errorf("source %s changed from %s to %s", sourceRef, shortOID(action.ExpectedSource), shortOID(currentSource))
	}

	targetRef := remoteTrackingRef(action.Target.Name, action.Target.Branch)
	currentTarget, err := git.ResolveRevision(repoPath, targetRef)
	if err != nil {
		return fmt.Errorf("resolve target %s: %w", targetRef, err)
	}
	if strings.TrimSpace(currentTarget) != strings.TrimSpace(action.ExpectedOldTarget) {
		return fmt.Errorf("target %s changed from %s to %s", targetRef, shortOID(action.ExpectedOldTarget), shortOID(currentTarget))
	}
	return nil
}

func remoteTrackingRef(remote, branch string) string {
	return "refs/remotes/" + strings.TrimSpace(remote) + "/" + strings.TrimSpace(branch)
}

func shortOID(oid string) string {
	oid = strings.TrimSpace(oid)
	if len(oid) <= 12 {
		return oid
	}
	return oid[:12]
}
