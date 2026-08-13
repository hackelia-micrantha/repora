package managedartifact

import (
	"fmt"

	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/transport"
)

const (
	canonicalREADMEObserverRemote = "canonical"
	canonicalREADMEObserverHead   = "canonical/HEAD"
)

type readmeGitClient interface {
	EnsureMirror(path, canonicalURL string) error
	ConfigureRemote(repoPath, name, url string) error
	Fetch(repoPath, name string) error
	SetRemoteHead(repoPath, name string) error
	ResolveRemoteHeadBranch(repoPath, remote string) (string, error)
	ResolveRevision(repoPath, rev string) (string, error)
	ReadTreeEntry(repoPath, rev, treePath string) (gitwrap.TreeEntry, bool, error)
	ReadBlobBounded(repoPath, oid string, maxBytes int64) ([]byte, error)
}

type resolveCanonicalFunc func(config.Endpoint) (transport.ResolvedRemote, error)
type mirrorPathFunc func(string) (string, error)

// GitREADMEObserver reads the canonical default-branch README through Repora's
// existing bare Git cache. It may create/update local cache state and fetch the
// canonical remote, but it never writes repository content, commits, or pushes.
type GitREADMEObserver struct {
	git              readmeGitClient
	resolveCanonical resolveCanonicalFunc
	mirrorPath       mirrorPathFunc
}

// NewGitREADMEObserver returns the production HTTPS-backed canonical observer.
func NewGitREADMEObserver() *GitREADMEObserver {
	resolver := transport.DefaultResolver(transport.HTTPS)
	return newGitREADMEObserver(gitwrap.Client{}, resolver.Resolve, gitwrap.MirrorPath)
}

func newGitREADMEObserver(git readmeGitClient, resolve resolveCanonicalFunc, mirrorPath mirrorPathFunc) *GitREADMEObserver {
	return &GitREADMEObserver{git: git, resolveCanonical: resolve, mirrorPath: mirrorPath}
}

// ObserveREADME fetches and binds README state to the exact canonical default
// branch and commit currently present in Repora's canonical cache.
func (o *GitREADMEObserver) ObserveREADME(repo config.Repo) (READMEObservation, error) {
	if o == nil || o.git == nil || o.resolveCanonical == nil || o.mirrorPath == nil {
		return READMEObservation{}, fmt.Errorf("git README observer is not fully configured")
	}
	if err := validatePlannerRepositories([]config.Repo{repo}); err != nil {
		return READMEObservation{}, err
	}

	canonical, err := o.resolveCanonical(repo.Canonical)
	if err != nil {
		return READMEObservation{}, fmt.Errorf("resolve canonical for repo %q: %w", repo.ID, err)
	}
	cachePath, err := o.mirrorPath(repo.DurableID())
	if err != nil {
		return READMEObservation{}, fmt.Errorf("resolve canonical cache path for repo %q: %w", repo.ID, err)
	}
	if err := o.git.EnsureMirror(cachePath, canonical.URL); err != nil {
		return READMEObservation{}, fmt.Errorf("prepare canonical cache for repo %q: %w", repo.ID, err)
	}
	if err := o.git.ConfigureRemote(cachePath, canonicalREADMEObserverRemote, canonical.URL); err != nil {
		return READMEObservation{}, fmt.Errorf("configure canonical cache remote for repo %q: %w", repo.ID, err)
	}
	if err := o.git.Fetch(cachePath, canonicalREADMEObserverRemote); err != nil {
		return READMEObservation{}, fmt.Errorf("fetch canonical README state for repo %q: %w", repo.ID, err)
	}
	if err := o.git.SetRemoteHead(cachePath, canonicalREADMEObserverRemote); err != nil {
		return READMEObservation{}, fmt.Errorf("resolve canonical default branch for repo %q: %w", repo.ID, err)
	}

	branch, err := o.git.ResolveRemoteHeadBranch(cachePath, canonicalREADMEObserverRemote)
	if err != nil {
		return READMEObservation{}, fmt.Errorf("read canonical default branch for repo %q: %w", repo.ID, err)
	}
	baseOID, err := o.git.ResolveRevision(cachePath, canonicalREADMEObserverHead)
	if err != nil {
		return READMEObservation{}, fmt.Errorf("read canonical base OID for repo %q: %w", repo.ID, err)
	}
	observation := READMEObservation{Branch: branch, BaseOID: baseOID}

	entry, present, err := o.git.ReadTreeEntry(cachePath, canonicalREADMEObserverHead, READMEPath)
	if err != nil {
		return READMEObservation{}, fmt.Errorf("read canonical README tree entry for repo %q: %w", repo.ID, err)
	}
	if !present {
		return observation, nil
	}
	if entry.Type != "blob" || !validGitMode(entry.Mode) {
		return READMEObservation{}, fmt.Errorf("repo %q canonical README.md must be a regular Git blob with mode 100644 or 100755; got type=%q mode=%q", repo.ID, entry.Type, entry.Mode)
	}
	content, err := o.git.ReadBlobBounded(cachePath, entry.OID, int64(MaxTextBytes))
	if err != nil {
		return READMEObservation{}, fmt.Errorf("read canonical README blob for repo %q: %w", repo.ID, err)
	}
	observation.Present = true
	observation.Mode = entry.Mode
	observation.Content = content
	return observation, nil
}
