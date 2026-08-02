// Package executor owns reconciliation side effects.
//
// It consumes validated, versioned plan artifacts without inspecting repository
// status or re-deciding reconciliation policy.
package executor

import (
	"fmt"
	"strings"

	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
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
	Index    int
	Action   plan.PlannedAction
	Outcome  Outcome
	AfterOID string
	Stale    bool
	Error    string
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

// Preflight validates an exact artifact and verifies every expected source and
// target reference without mutating a remote.
func Preflight(repoPath string, artifact planartifact.Artifact, git Git) (Result, error) {
	planned, err := onePlan(artifact)
	if err != nil {
		return Result{}, err
	}
	return preflightPlan(repoPath, planned, git)
}

// Execute validates and consumes exactly one repository plan from a versioned
// artifact. Invalid or stale artifacts fail closed before mutation.
func Execute(repoPath string, artifact planartifact.Artifact, git Git) (Result, error) {
	planned, err := onePlan(artifact)
	if err != nil {
		return Result{}, err
	}
	return executePlan(repoPath, planned, git)
}

func onePlan(artifact planartifact.Artifact) (plan.ReconciliationPlan, error) {
	plans, err := artifact.Plans()
	if err != nil {
		return plan.ReconciliationPlan{}, fmt.Errorf("validate plan artifact: %w", err)
	}
	if len(plans) != 1 {
		return plan.ReconciliationPlan{}, fmt.Errorf("plan artifact requires exactly one repository, got %d", len(plans))
	}
	return plans[0], nil
}

// executePlan remains the plan-level execution boundary used by focused tests.
// Artifact callers reach it only after strict artifact parsing and conversion.
func executePlan(repoPath string, planned plan.ReconciliationPlan, git Git) (Result, error) {
	result, err := preflightPlan(repoPath, planned, git)
	if err != nil {
		return result, err
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
		result.Actions[i].AfterOID = action.ExpectedSource
	}
	return result, nil
}

func preflightPlan(repoPath string, planned plan.ReconciliationPlan, git Git) (Result, error) {
	result := Result{Actions: make([]ActionResult, len(planned.Actions))}
	for i, action := range planned.Actions {
		result.Actions[i] = ActionResult{Index: i, Action: action, Outcome: OutcomeSkipped}
	}

	for i, action := range planned.Actions {
		if err := validateAction(action); err != nil {
			return failPreflight(result, i, false, fmt.Errorf("validate action %d: %w", i, err))
		}
	}
	for i, action := range planned.Actions {
		if err := validateCurrentRefs(repoPath, action, git); err != nil {
			return failPreflight(result, i, true, fmt.Errorf("stale action %d: %w", i, err))
		}
	}
	return result, nil
}

func failPreflight(result Result, index int, stale bool, err error) (Result, error) {
	result.Actions[index].Outcome = OutcomeFailed
	result.Actions[index].Stale = stale
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
