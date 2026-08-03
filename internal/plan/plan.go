// Package plan owns deterministic reconciliation decisions.
//
// It consumes configured repository identity plus observed repository state and
// produces in-memory actions. It must not perform Git mutations.
package plan

import (
	"fmt"
	"strings"

	"repoctl/internal/config"
	"repoctl/internal/refpolicy"
	"repoctl/internal/status"
)

const (
	OutputKind    = "repora.plan"
	OutputVersion = 1
)

// Output is the stabilized v1 CLI compatibility view. It is projected from
// exact reconciliation plans and must never make independent mutation
// decisions.
type Output struct {
	Kind    string     `json:"kind"`
	Version int        `json:"version"`
	Plan    []RepoPlan `json:"plan"`
}

type RepoPlan struct {
	ID      string   `json:"id"`
	UID     string   `json:"uid"`
	Actions []Action `json:"actions"`
}

type Action struct {
	Type        string `json:"type"`
	Target      string `json:"target"`
	Behind      int    `json:"behind"`
	Destructive bool   `json:"destructive"`
}

func NewOutput(plans []ReconciliationPlan, results []status.Result) Output {
	count := len(plans)
	if len(results) < count {
		count = len(results)
	}
	out := Output{Kind: OutputKind, Version: OutputVersion, Plan: make([]RepoPlan, 0, count)}
	for i := 0; i < count; i++ {
		out.Plan = append(out.Plan, NewRepoPlan(plans[i], results[i]))
	}
	return out
}

func NewRepoPlan(planned ReconciliationPlan, result status.Result) RepoPlan {
	repoPlan := RepoPlan{ID: planned.ID, UID: planned.UID, Actions: []Action{}}
	for _, action := range planned.Actions {
		repoPlan.Actions = append(repoPlan.Actions, Action{
			Type:        "PUSH_MIRROR",
			Target:      action.Target.Provider,
			Behind:      result.Behind,
			Destructive: action.Force,
		})
	}
	return repoPlan
}

type ActionType string

const ActionPushBranch ActionType = "PUSH_BRANCH"

type Remote struct {
	Provider string
	Name     string
	Branch   string
}

type PlannedAction struct {
	Type              ActionType
	Source            Remote
	Target            Remote
	Force             bool
	ExpectedSource    string
	ExpectedOldTarget string
	Reason            string
}

type ReconciliationPlan struct {
	ID      string
	UID     string
	Actions []PlannedAction
}

type Observation struct {
	CanonicalBranch  string
	CanonicalHeadOID string
	MirrorBranch     string
	MirrorHeadOID    string
}

// RequiresMirrorHeadObservation reports whether planning needs the current
// mirror head to construct a forced action and its lease.
func RequiresMirrorHeadObservation(result status.Result) bool {
	return result.State == status.StateAhead || result.State == status.StateDiverged
}

// RequiresRefObservation reports whether reconciliation may produce a mutation
// action whose source and target refs must be captured for stale-plan checks.
func RequiresRefObservation(result status.Result) bool {
	return result.State == status.StateBehind || RequiresMirrorHeadObservation(result)
}

func Reconcile(repo config.Repo, result status.Result, observed Observation, force bool) (ReconciliationPlan, error) {
	repoPlan := ReconciliationPlan{ID: repo.ID, UID: repo.DurableID(), Actions: []PlannedAction{}}

	policy, err := repo.EffectiveRefPolicy()
	if err != nil {
		return repoPlan, fmt.Errorf("invalid ref policy for repo %q: %w", repo.ID, err)
	}
	decision, err := policy.Decide(refpolicy.Relationship(result.State))
	if err != nil {
		return repoPlan, fmt.Errorf("unsupported state %q for repo %q: %w", result.State, repo.ID, err)
	}
	if !decision.Action {
		return repoPlan, nil
	}

	if len(repo.Mirrors) != 1 {
		return repoPlan, fmt.Errorf("repo %q requires exactly one configured mirror, got %d", repo.ID, len(repo.Mirrors))
	}
	if strings.TrimSpace(repo.Canonical.Provider) == "" || strings.TrimSpace(repo.Mirrors[0].Provider) == "" {
		return repoPlan, fmt.Errorf("repo %q requires canonical and mirror providers", repo.ID)
	}

	sourceBranch := strings.TrimSpace(observed.CanonicalBranch)
	targetBranch := strings.TrimSpace(observed.MirrorBranch)
	if targetBranch == "" {
		targetBranch = sourceBranch
	}
	if sourceBranch == "" || targetBranch == "" {
		return repoPlan, fmt.Errorf("repo %q requires resolved canonical and mirror branches", repo.ID)
	}

	expectedSource := strings.TrimSpace(observed.CanonicalHeadOID)
	expectedTarget := strings.TrimSpace(observed.MirrorHeadOID)
	if expectedSource == "" || expectedTarget == "" {
		return repoPlan, fmt.Errorf("repo %q requires observed canonical and mirror heads", repo.ID)
	}

	action := PlannedAction{
		Type: ActionPushBranch,
		Source: Remote{
			Provider: repo.Canonical.Provider,
			Name:     "canonical",
			Branch:   sourceBranch,
		},
		Target: Remote{
			Provider: repo.Mirrors[0].Provider,
			Name:     "mirror",
			Branch:   targetBranch,
		},
		Force:             decision.Force,
		ExpectedSource:    expectedSource,
		ExpectedOldTarget: expectedTarget,
		Reason:            decision.Reason,
	}

	repoPlan.Actions = append(repoPlan.Actions, action)
	if decision.Force && !force {
		return repoPlan, fmt.Errorf("repo %q is %s; rerun with --force to overwrite mirror default branch using a lease against %s", repo.ID, result.State, shortOID(action.ExpectedOldTarget))
	}
	return repoPlan, nil
}

func shortOID(oid string) string {
	if len(oid) <= 12 {
		return oid
	}
	return oid[:12]
}
