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

	path, err := gitwrap.MirrorPath(repo.DurableID())
	if err != nil {
		return result, err
	}

	srcBranch, err := git.ResolveRemoteHeadBranch(path, "canonical")
	if err != nil {
		return result, fmt.Errorf("resolve canonical HEAD for repo %q: %w", repo.ID, err)
	}
	dstBranch, err := git.ResolveRemoteHeadBranch(path, "mirror")
	if err != nil {
		return result, fmt.Errorf("resolve mirror HEAD for repo %q: %w", repo.ID, err)
	}
	if dstBranch == "" {
		dstBranch = srcBranch
	}

	observation := plan.Observation{CanonicalBranch: srcBranch, MirrorBranch: dstBranch}
	if plan.RequiresMirrorHeadObservation(st) {
		dstRef := "refs/remotes/mirror/" + dstBranch
		expectedOldOID, err := git.ResolveRevision(path, dstRef)
		if err != nil {
			return result, fmt.Errorf("resolve mirror branch for repo %q: %w", repo.ID, err)
		}
		observation.MirrorHeadOID = expectedOldOID
	}

	planned, planErr := plan.Reconcile(repo, st, observation, force)
	for _, action := range planned.Actions {
		result.Actions = append(result.Actions, Action{
			Type:              string(action.Type),
			Source:            action.Source.Name + "/" + action.Source.Branch,
			Target:            action.Target.Provider + "/" + action.Target.Branch,
			Force:             action.Force,
			ExpectedOldTarget: action.ExpectedOldTarget,
		})
	}
	if planErr != nil {
		return result, planErr
	}
	if dryRun || len(planned.Actions) == 0 {
		return result, nil
	}

	executed, err := executor.Execute(path, planned, git)
	if err != nil {
		return result, fmt.Errorf("execute plan for repo %q: %w", repo.ID, err)
	}
	result.Applied = len(executed.Actions) > 0
	return result, nil
}
