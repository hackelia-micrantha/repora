package apply

import (
	"fmt"
	"strings"

	"repoctl/internal/config"
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
	ResolveRemoteHeadBranch(repoPath, remote string) (string, error)
	ResolveRevision(repoPath, rev string) (string, error)
	PushBranch(repoPath, remote, srcRef, dstBranch string) error
	ForcePushBranchWithLease(repoPath, remote, srcRef, dstBranch, expectedOldOID string) error
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
		dstRef := remoteTrackingRef("mirror", dstBranch)
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

	action := planned.Actions[0]
	srcRef := remoteTrackingRef(action.Source.Name, action.Source.Branch)
	if action.Force {
		if err := git.ForcePushBranchWithLease(path, action.Target.Name, srcRef, action.Target.Branch, action.ExpectedOldTarget); err != nil {
			return result, fmt.Errorf("force push mirror branch with lease for repo %q: %w", repo.ID, err)
		}
	} else {
		if err := git.PushBranch(path, action.Target.Name, srcRef, action.Target.Branch); err != nil {
			return result, fmt.Errorf("push mirror branch for repo %q: %w", repo.ID, err)
		}
	}
	result.Applied = true
	return result, nil
}

func remoteTrackingRef(remote, branch string) string {
	return "refs/remotes/" + remote + "/" + strings.TrimSpace(branch)
}
