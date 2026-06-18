package apply

import (
	"fmt"
	"strings"

	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
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

func IsUnsafe(result status.Result) bool {
	return result.State == status.StateAhead || result.State == status.StateDiverged
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

	srcRef := remoteTrackingRef("canonical", srcBranch)
	dstRef := remoteTrackingRef("mirror", dstBranch)
	action := Action{
		Type:   "PUSH_BRANCH",
		Source: "canonical/" + srcBranch,
		Target: repo.Mirrors[0].Provider + "/" + dstBranch,
	}

	switch st.State {
	case status.StateEqual:
		return result, nil
	case status.StateBehind:
		result.Actions = append(result.Actions, action)
		if dryRun {
			return result, nil
		}
		if err := git.PushBranch(path, "mirror", srcRef, dstBranch); err != nil {
			return result, fmt.Errorf("push mirror branch for repo %q: %w", repo.ID, err)
		}
		result.Applied = true
		return result, nil
	case status.StateAhead, status.StateDiverged:
		action.Force = true
		expectedOldOID, err := git.ResolveRevision(path, dstRef)
		if err != nil {
			return result, fmt.Errorf("resolve mirror branch for repo %q: %w", repo.ID, err)
		}
		action.ExpectedOldTarget = expectedOldOID
		result.Actions = append(result.Actions, action)
		if !force {
			return result, fmt.Errorf("repo %q is %s; rerun with --force to overwrite mirror default branch using a lease against %s", repo.ID, st.State, shortOID(expectedOldOID))
		}
		if dryRun {
			return result, nil
		}
		if err := git.ForcePushBranchWithLease(path, "mirror", srcRef, dstBranch, expectedOldOID); err != nil {
			return result, fmt.Errorf("force push mirror branch with lease for repo %q: %w", repo.ID, err)
		}
		result.Applied = true
		return result, nil
	default:
		return result, fmt.Errorf("unsupported state %q for repo %q", st.State, repo.ID)
	}
}

func remoteTrackingRef(remote, branch string) string {
	return "refs/remotes/" + remote + "/" + strings.TrimSpace(branch)
}

func shortOID(oid string) string {
	if len(oid) <= 12 {
		return oid
	}
	return oid[:12]
}
