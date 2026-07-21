package apply

import (
	"fmt"

	"repoctl/internal/config"
	"repoctl/internal/executor"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/plan"
	"repoctl/internal/status"
)

type Output struct {
	Results []Result `json:"results"`
}

type Result struct {
	ID      string       `json:"id"`
	UID     string       `json:"uid"`
	State   status.State `json:"state"`
	Applied bool         `json:"applied"`
	DryRun  bool         `json:"dry_run"`
	Actions []Action     `json:"actions"`
	Error   string       `json:"error,omitempty"`
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
	ResolveRevision(repoPath, rev string) (string, error)
}

type builtPlan struct {
	path    string
	planned plan.ReconciliationPlan
	actions []Action
}

// IsUnsafe is retained for callers that need to identify states requiring a
// mirror-head observation. The planner owns that classification.
func IsUnsafe(result status.Result) bool {
	return plan.RequiresMirrorHeadObservation(result)
}

func Execute(repo config.Repo, st status.Result, git Git, force bool, dryRun bool) (Result, error) {
	result := Result{
		ID:      repo.ID,
		UID:     repo.DurableID(),
		State:   st.State,
		DryRun:  dryRun,
		Actions: []Action{},
	}

	built, err := buildPlan(repo, st, git, force)
	result.Actions = built.actions
	if err != nil {
		return result, err
	}
	if dryRun || len(built.planned.Actions) == 0 {
		return result, nil
	}

	executed, err := executor.Execute(built.path, built.planned, git)
	if err != nil {
		return result, fmt.Errorf("execute plan for repo %q: %w", repo.ID, err)
	}
	result.Applied = executed.AllApplied()
	return result, nil
}

// buildPlan is the single observation-to-plan path used by both dry-run and
// real apply. Compatibility actions are projected from that exact plan once;
// execution receives the same in-memory plan without rebuilding decisions.
func buildPlan(repo config.Repo, st status.Result, git Git, force bool) (builtPlan, error) {
	built := builtPlan{actions: []Action{}}

	path, err := gitwrap.MirrorPath(repo.DurableID())
	if err != nil {
		return built, err
	}
	built.path = path

	srcBranch, err := git.ResolveRemoteHeadBranch(path, "canonical")
	if err != nil {
		return built, fmt.Errorf("resolve canonical HEAD for repo %q: %w", repo.ID, err)
	}
	dstBranch, err := git.ResolveRemoteHeadBranch(path, "mirror")
	if err != nil {
		return built, fmt.Errorf("resolve mirror HEAD for repo %q: %w", repo.ID, err)
	}
	if dstBranch == "" {
		dstBranch = srcBranch
	}

	observation := plan.Observation{CanonicalBranch: srcBranch, MirrorBranch: dstBranch}
	if plan.RequiresMirrorHeadObservation(st) {
		dstRef := "refs/remotes/mirror/" + dstBranch
		expectedOldOID, err := git.ResolveRevision(path, dstRef)
		if err != nil {
			return built, fmt.Errorf("resolve mirror branch for repo %q: %w", repo.ID, err)
		}
		observation.MirrorHeadOID = expectedOldOID
	}

	built.planned, err = plan.Reconcile(repo, st, observation, force)
	built.actions = compatibilityActions(built.planned)
	if err != nil {
		return built, err
	}
	return built, nil
}

func compatibilityActions(planned plan.ReconciliationPlan) []Action {
	actions := make([]Action, 0, len(planned.Actions))
	for _, action := range planned.Actions {
		actions = append(actions, Action{
			Type:              string(action.Type),
			Source:            action.Source.Name + "/" + action.Source.Branch,
			Target:            action.Target.Provider + "/" + action.Target.Branch,
			Force:             action.Force,
			ExpectedOldTarget: action.ExpectedOldTarget,
		})
	}
	return actions
}
