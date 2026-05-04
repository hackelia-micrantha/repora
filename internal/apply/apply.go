package apply

import (
	"fmt"

	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/status"
)

type Output struct {
	Apply []RepoApply `json:"apply"`
}

type RepoApply struct {
	ID      string   `json:"id"`
	Actions []Action `json:"actions"`
}

type Action struct {
	Type        string `json:"type"`
	Target      string `json:"target"`
	Forced      bool   `json:"forced"`
	Destructive bool   `json:"destructive"`
}

type Git interface {
	SyncMirrorFromRemote(repoPath, remote string) error
	PushMirror(repoPath, remote string) error
}

func IsUnsafe(result status.Result) bool {
	return result.State == status.StateAhead || result.State == status.StateDiverged
}

func Execute(repo config.Repo, result status.Result, git Git, force bool) (RepoApply, error) {
	repoApply := RepoApply{
		ID:      repo.ID,
		Actions: []Action{},
	}

	if result.State == status.StateEqual {
		return repoApply, nil
	}
	if IsUnsafe(result) && !force {
		return RepoApply{}, fmt.Errorf("repo %q mirror state is %s; rerun apply with --force to overwrite mirror from canonical", repo.ID, result.State)
	}
	if result.State != status.StateBehind && !IsUnsafe(result) {
		return RepoApply{}, fmt.Errorf("repo %q has unsupported state %s", repo.ID, result.State)
	}

	path, err := gitwrap.MirrorPath(repo.ID)
	if err != nil {
		return RepoApply{}, err
	}
	if err := git.SyncMirrorFromRemote(path, "canonical"); err != nil {
		return RepoApply{}, err
	}
	if err := git.PushMirror(path, "mirror"); err != nil {
		return RepoApply{}, err
	}

	repoApply.Actions = append(repoApply.Actions, Action{
		Type:        "PUSH_MIRROR",
		Target:      repo.Mirrors[0].Provider,
		Forced:      force,
		Destructive: IsUnsafe(result),
	})
	return repoApply, nil
}
