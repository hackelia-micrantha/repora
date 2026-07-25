// Package plan owns deterministic reconciliation decisions.
//
// It consumes configured repository identity plus observed repository state and
// produces in-memory actions. It must not perform Git mutations.
package plan

import (
	"fmt"
	"strings"

	"repoctl/internal/config"
	"repoctl/internal/status"
)

// Output is the existing user-facing plan command model. Its serialized shape
// is intentionally kept separate from the in-memory reconciliation model below.
type Output struct {
	Plan []RepoPlan `json:"plan"`
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

func NewOutput(spec config.Spec, results []status.Result, ok []bool) Output {
	out := Output{Plan: make([]RepoPlan, 0, len(spec.Repos))}
	for i, repo := range spec.Repos {
		if !ok[i] {
			continue
		}
		out.Plan = append(out.Plan, NewRepoPlan(repo, results[i]))
	}
	return out
}

func NewRepoPlan(repo config.Repo, result status.Result) RepoPlan {
	repoPlan := RepoPlan{ID: repo.ID, UID: repo.DurableID(), Actions: []Action{}}
	if result.State == status.StateBehind {
		repoPlan.Actions = append(repoPlan.Actions, Action{
			Type:        "PUSH_MIRROR",
			Target:      repo.Mirrors[0].Provider,
			Behind:      result.Behind,
			Destructive: false,
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

	if err := validatePlanningTopology(repo); err != nil {
		return repoPlan, err
	}

	switch result.State {
	case status.StateEqual:
		return repoPlan, nil
	case status.StateBehind, status.StateAhead, status.StateDiverged:
		// Continue below.
	default:
		return repoPlan, fmt.Errorf("unsupported state %q for repo %q", result.State, repo.ID)
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
		ExpectedSource:    expectedSource,
		ExpectedOldTarget: expectedTarget,
		Reason:            fmt.Sprintf("mirror is %s", strings.ToLower(string(result.State))),
	}

	// The supported topology has exactly one mirror, so a reconciliation plan
	// contains at most one action. This is the stable action-ordering contract
	// until multi-mirror planning is introduced explicitly.
	if result.State == status.StateBehind {
		repoPlan.Actions = append(repoPlan.Actions, action)
		return repoPlan, nil
	}

	action.Force = true
	repoPlan.Actions = append(repoPlan.Actions, action)
	if !force {
		return repoPlan, fmt.Errorf("repo %q is %s; rerun with --force to overwrite mirror default branch using a lease against %s", repo.ID, result.State, shortOID(action.ExpectedOldTarget))
	}
	return repoPlan, nil
}

func validatePlanningTopology(repo config.Repo) error {
	if strings.TrimSpace(repo.ID) == "" {
		return fmt.Errorf("planner requires a non-empty repo id")
	}
	if repo.Canonical.Provider != "gitlab" {
		return fmt.Errorf("repo %q has unsupported canonical provider %q: planner supports gitlab", repo.ID, repo.Canonical.Provider)
	}
	if len(repo.Mirrors) != 1 {
		return fmt.Errorf("repo %q has ambiguous mirror topology: planner requires exactly one mirror, got %d", repo.ID, len(repo.Mirrors))
	}
	mirrorProvider := repo.Mirrors[0].Provider
	if mirrorProvider != "github" && mirrorProvider != "gitlab" {
		return fmt.Errorf("repo %q has unsupported mirror provider %q: planner supports github and gitlab", repo.ID, mirrorProvider)
	}
	if repo.Mode != "" && repo.Mode != "mirror" {
		return fmt.Errorf("repo %q has unsupported mode %q: planner supports mirror", repo.ID, repo.Mode)
	}
	return nil
}

func shortOID(oid string) string {
	if len(oid) <= 12 {
		return oid
	}
	return oid[:12]
}
