package managedartifact

import (
	"bytes"
	"fmt"

	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
)

const managedREADMECommitMessage = "chore: update managed README"

type localCommitGit interface {
	WriteBlob(repoPath string, content []byte) (string, error)
	BuildTreeWithRootBlob(repoPath, baseOID, treePath, mode, blobOID string) (string, error)
	CreateCommitObject(repoPath, treeOID, parentOID, message string) (string, error)
	ChangedPaths(repoPath, baseOID, commitOID string) ([]string, error)
	ReadTreeEntry(repoPath, rev, treePath string) (gitwrap.TreeEntry, bool, error)
	ReadBlobBounded(repoPath, oid string, maxBytes int64) ([]byte, error)
}

type localCommitMirrorPath func(string) (string, error)

// PreparedCommit is local execution evidence for one otherwise-unreferenced
// candidate commit. Creating it does not update any Git ref or remote state.
type PreparedCommit struct {
	UID       string
	ID        string
	BaseOID   string
	TreeOID   string
	CommitOID string
}

// CommitPreparer constructs reviewed README commits in Repora's bare cache.
// The Git dependency intentionally exposes no ref-update or push operation.
type CommitPreparer struct {
	git        localCommitGit
	mirrorPath localCommitMirrorPath
}

func NewCommitPreparer() *CommitPreparer {
	return newCommitPreparer(gitwrap.Client{}, gitwrap.MirrorPath)
}

func newCommitPreparer(git localCommitGit, mirrorPath localCommitMirrorPath) *CommitPreparer {
	return &CommitPreparer{git: git, mirrorPath: mirrorPath}
}

// Prepare preflights the exact plan, then creates one unreachable local commit
// per planned repository. No branch, tag, remote ref, or worktree is changed.
func (p *CommitPreparer) Prepare(spec config.Spec, plan Plan, observer READMEObserver) ([]PreparedCommit, error) {
	if p == nil || p.git == nil || p.mirrorPath == nil {
		return nil, fmt.Errorf("managed README commit preparer is not fully configured")
	}
	if err := PreflightPlan(spec, plan, observer); err != nil {
		return nil, err
	}
	prepared := make([]PreparedCommit, 0, len(plan.Repositories))
	for _, repo := range plan.Repositories {
		cachePath, err := p.mirrorPath(repo.UID)
		if err != nil {
			return nil, fmt.Errorf("repo %q: resolve commit cache path: %w", repo.ID, err)
		}
		action := repo.Actions[0]
		desired := []byte(*action.Desired.Content)
		blobOID, err := p.git.WriteBlob(cachePath, desired)
		if err != nil {
			return nil, fmt.Errorf("repo %q: write desired README blob: %w", repo.ID, err)
		}
		treeOID, err := p.git.BuildTreeWithRootBlob(cachePath, repo.BaseOID, READMEPath, action.Desired.Mode, blobOID)
		if err != nil {
			return nil, fmt.Errorf("repo %q: build isolated README tree: %w", repo.ID, err)
		}
		commitOID, err := p.git.CreateCommitObject(cachePath, treeOID, repo.BaseOID, managedREADMECommitMessage)
		if err != nil {
			return nil, fmt.Errorf("repo %q: create isolated README commit: %w", repo.ID, err)
		}
		if err := verifyPreparedCommit(p.git, cachePath, repo, commitOID, desired); err != nil {
			return nil, err
		}
		prepared = append(prepared, PreparedCommit{
			UID:       repo.UID,
			ID:        repo.ID,
			BaseOID:   repo.BaseOID,
			TreeOID:   treeOID,
			CommitOID: commitOID,
		})
	}
	return prepared, nil
}

func verifyPreparedCommit(git localCommitGit, cachePath string, planned RepositoryPlan, commitOID string, desired []byte) error {
	paths, err := git.ChangedPaths(cachePath, planned.BaseOID, commitOID)
	if err != nil {
		return fmt.Errorf("repo %q: verify isolated commit changed paths: %w", planned.ID, err)
	}
	if len(paths) != 1 || paths[0] != READMEPath {
		return fmt.Errorf("repo %q: isolated commit changed paths %q; want exactly %q", planned.ID, paths, READMEPath)
	}
	entry, present, err := git.ReadTreeEntry(cachePath, commitOID, READMEPath)
	if err != nil {
		return fmt.Errorf("repo %q: verify isolated commit README entry: %w", planned.ID, err)
	}
	if !present || entry.Type != "blob" || entry.Mode != planned.Actions[0].Desired.Mode {
		return fmt.Errorf("repo %q: isolated commit README entry does not match reviewed mode", planned.ID)
	}
	content, err := git.ReadBlobBounded(cachePath, entry.OID, int64(MaxTextBytes))
	if err != nil {
		return fmt.Errorf("repo %q: verify isolated commit README blob: %w", planned.ID, err)
	}
	if !bytes.Equal(content, desired) || DigestSHA256(content) != planned.Actions[0].Desired.SHA256 {
		return fmt.Errorf("repo %q: isolated commit README content does not match reviewed desired content", planned.ID)
	}
	return nil
}
