// Package plan owns deterministic reconciliation decisions.
//
// It consumes configured repository identity plus observed repository state and
// produces in-memory actions. It must not perform Git mutations. The apply
// package temporarily remains responsible for resolving local execution details
// and executing these actions until later issue #22 slices introduce an
// executor boundary.
package plan

import (
	"fmt"
	"strings"

	"repoctl/internal/config"
	"repoctl/internal/status"
)

type Output struct {
	Plan []RepoPlan `json:"plan"`
}

type RepoPlan struct {
	ID      string   `json:"id"`
	UID     string   `json:"uid"`
	Actions []Action `json:"actions"`
}

type ActionType string

const ActionPushBranch ActionType = "PUSH_BRANCH"

type Remote struct {
	Provider string
	Name     string
	Branch   string
}

type Action struct {
	Type              ActionType
	Source            Remote
	Target            Remote
	Force             bool
	ExpectedOldTarget string
	Reason            string
}

type Observation struct {
	CanonicalBranch string
	MirrorBranch    string
	MirrorHeadOID   string
}

func NewOutput(spec config.Spec, results []status.Result, ok []bool) Output {
	out := Output{Plan: make([]RepoPlan, 0, len(spec.Repos))}
	for i, repo := range spec.Repos {
		if !ok[i] {
			continue
		}
		repoPlan, err := NewRepoPlan(repo, results[i], Observation{}, false)
		if err != nil {
			continue
		}
		out.Plan = append(out.Plan, repoPlan)
	}
	return out
}

func NewRepoPlan(repo config.Repo, result status.Result, observed Observation, force bool) (RepoPlan, error) {
	repoPlan := RepoPlan{
		ID:      repo.ID,
		UID:     repo.DurableID(),
		Actions: []Action{},
	}

	switch result.State {
	case status.StateEqual:
		return repoPlan, nil
	case status.StateBehind, status.StateAhead, status.StateDiverged:
		// Continue below.
	default:
		return repoPlan, fmt.Errorf("unsupported state %q for repo %q", result.State, repo.ID)
	}

	if len(repo.Mirrors) == 0 {
		return repoPlan, fmt.Errorf("repo %q has no configured mirror", repo.ID)
	}

	sourceBranch := strings.TrimSpace(observed.CanonicalBranch)
	targetBranch := strings.TrimSpace(observed.MirrorBranch)
	if targetBranch == "" {
		targetBranch = sourceBranch
	}
	if sourceBranch == "" || targetBranch == "" {
		return repoPlan, fmt.Errorf("repo %q requires resolved canonical and mirror branches", repo.ID)
	}

	action := Action{
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
		Reason: fmt.Sprintf("mirror is %s", strings.ToLower(string(result.State))),
	}

	if result.State == status.StateBehind {
		repoPlan.Actions = append(repoPlan.Actions, action)
		return repoPlan, nil
	}

	action.Force = true
	action.ExpectedOldTarget = strings.TrimSpace(observed.MirrorHeadOID)
	if action.ExpectedOldTarget == "" {
		return repoPlan, fmt.Errorf("repo %q requires observed mirror head for forced update", repo.ID)
	}
	repoPlan.Actions = append(repoPlan.Actions, action)
	if !force {
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
