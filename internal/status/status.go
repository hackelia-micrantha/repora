package status

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/transport"
)

const (
	OutputKind    = "repora.status"
	OutputVersion = 2
)

type State string

const (
	StateEqual    State = "EQUAL"
	StateBehind   State = "BEHIND"
	StateAhead    State = "AHEAD"
	StateDiverged State = "DIVERGED"
	StateError    State = "ERROR"
)

// Result is the single-mirror reconciliation observation consumed by plan and
// apply. Multi-mirror status uses RepositoryResult instead.
type Result struct {
	ID        string `json:"id"`
	UID       string `json:"uid"`
	State     State  `json:"state"`
	Ahead     int    `json:"ahead"`
	Behind    int    `json:"behind"`
	Canonical string `json:"-"`
	Mirror    string `json:"-"`
}

type Output struct {
	Kind    string             `json:"kind"`
	Version int                `json:"version"`
	Repos   []RepositoryResult `json:"repos"`
}

type RepositoryResult struct {
	ID        string         `json:"id"`
	UID       string         `json:"uid"`
	Canonical RefResult      `json:"canonical"`
	Mirrors   []MirrorResult `json:"mirrors"`
	Error     string         `json:"error,omitempty"`
}

type RefResult struct {
	Ref    string `json:"ref"`
	Commit string `json:"commit"`
}

type MirrorResult struct {
	Target   string `json:"target"`
	Provider string `json:"provider"`
	Path     string `json:"path"`
	Ref      string `json:"ref"`
	Commit   string `json:"commit"`
	State    State  `json:"state"`
	Ahead    int    `json:"ahead"`
	Behind   int    `json:"behind"`
	Error    string `json:"error,omitempty"`
}

type Git interface {
	EnsureMirror(path, canonicalURL string) error
	ConfigureRemote(repoPath, name, url string) error
	Fetch(repoPath, name string) error
	SetRemoteHead(repoPath, name string) error
	RevListLeftRightCount(repoPath, left, right string) (string, error)
	RevParseShort(repoPath, rev string) (string, error)
}

// Check observes exactly one configured mirror for reconciliation planning.
func Check(repo config.Repo, git Git) (Result, error) {
	if len(repo.Mirrors) != 1 {
		return Result{}, fmt.Errorf("repo %q requires exactly one mirror for reconciliation, got %d", repo.ID, len(repo.Mirrors))
	}
	resolver := transport.DefaultResolver(transport.HTTPS)
	canonical, err := resolver.Resolve(repo.Canonical)
	if err != nil {
		return Result{}, fmt.Errorf("resolve canonical for repo %q: %w", repo.ID, err)
	}
	mirror, err := resolver.Resolve(repo.Mirrors[0])
	if err != nil {
		return Result{}, fmt.Errorf("resolve mirror for repo %q: %w", repo.ID, err)
	}

	path, err := gitwrap.MirrorPath(repo.DurableID())
	if err != nil {
		return Result{}, err
	}

	if err := git.EnsureMirror(path, canonical.URL); err != nil {
		return Result{}, err
	}
	if err := git.ConfigureRemote(path, "canonical", canonical.URL); err != nil {
		return Result{}, err
	}
	if err := git.ConfigureRemote(path, "mirror", mirror.URL); err != nil {
		return Result{}, err
	}
	if err := git.Fetch(path, "canonical"); err != nil {
		return Result{}, err
	}
	if err := git.Fetch(path, "mirror"); err != nil {
		return Result{}, err
	}
	if err := git.SetRemoteHead(path, "canonical"); err != nil {
		return Result{}, err
	}
	if err := git.SetRemoteHead(path, "mirror"); err != nil {
		return Result{}, err
	}

	counts, err := git.RevListLeftRightCount(path, "canonical/HEAD", "mirror/HEAD")
	if err != nil {
		return Result{}, err
	}
	left, right, err := ParseRevListCount(counts)
	if err != nil {
		return Result{}, err
	}
	result := InterpretDivergence(left, right)
	result.ID = repo.ID
	result.UID = repo.DurableID()

	canonicalCommit, err := git.RevParseShort(path, "canonical/HEAD")
	if err != nil {
		return Result{}, fmt.Errorf("resolve canonical commit evidence for repo %q: %w", repo.ID, err)
	}
	mirrorCommit, err := git.RevParseShort(path, "mirror/HEAD")
	if err != nil {
		return Result{}, fmt.Errorf("resolve mirror commit evidence for repo %q: %w", repo.ID, err)
	}
	result.Canonical = strings.TrimSpace(canonicalCommit)
	result.Mirror = strings.TrimSpace(mirrorCommit)
	return result, nil
}

// CheckAll observes every configured mirror independently for status output.
// Canonical setup is shared; mirror failures remain attached to their target and
// do not hide healthy mirror results.
func CheckAll(repo config.Repo, git Git) (RepositoryResult, error) {
	result := RepositoryResult{
		ID:        repo.ID,
		UID:       repo.DurableID(),
		Canonical: RefResult{Ref: "HEAD"},
		Mirrors:   []MirrorResult{},
	}
	if len(repo.Mirrors) == 0 {
		err := fmt.Errorf("repo %q requires at least one mirror", repo.ID)
		result.Error = err.Error()
		return result, err
	}

	resolver := transport.DefaultResolver(transport.HTTPS)
	canonical, err := resolver.Resolve(repo.Canonical)
	if err != nil {
		return failRepository(result, fmt.Errorf("resolve canonical for repo %q: %w", repo.ID, err))
	}
	path, err := gitwrap.MirrorPath(repo.DurableID())
	if err != nil {
		return failRepository(result, err)
	}
	if err := git.EnsureMirror(path, canonical.URL); err != nil {
		return failRepository(result, fmt.Errorf("prepare cache for repo %q: %w", repo.ID, err))
	}
	if err := git.ConfigureRemote(path, "canonical", canonical.URL); err != nil {
		return failRepository(result, fmt.Errorf("configure canonical for repo %q: %w", repo.ID, err))
	}
	if err := git.Fetch(path, "canonical"); err != nil {
		return failRepository(result, fmt.Errorf("fetch canonical for repo %q: %w", repo.ID, err))
	}
	if err := git.SetRemoteHead(path, "canonical"); err != nil {
		return failRepository(result, fmt.Errorf("resolve canonical HEAD for repo %q: %w", repo.ID, err))
	}
	canonicalCommit, err := git.RevParseShort(path, "canonical/HEAD")
	if err != nil {
		return failRepository(result, fmt.Errorf("resolve canonical commit evidence for repo %q: %w", repo.ID, err))
	}
	result.Canonical.Commit = strings.TrimSpace(canonicalCommit)

	var mirrorErrors []error
	for i, endpoint := range repo.Mirrors {
		mirrorResult, identityErr := newMirrorResult(endpoint)
		if identityErr != nil {
			mirrorResult.Error = identityErr.Error()
			result.Mirrors = append(result.Mirrors, mirrorResult)
			mirrorErrors = append(mirrorErrors, identityErr)
			continue
		}
		remoteName := "mirror-" + strconv.Itoa(i)
		if err := observeMirror(path, remoteName, endpoint, resolver, git, &mirrorResult); err != nil {
			mirrorResult.State = StateError
			mirrorResult.Error = err.Error()
			mirrorErrors = append(mirrorErrors, fmt.Errorf("%s: %w", mirrorResult.Target, err))
		}
		result.Mirrors = append(result.Mirrors, mirrorResult)
	}
	if len(mirrorErrors) > 0 {
		return result, errors.Join(mirrorErrors...)
	}
	return result, nil
}

func failRepository(result RepositoryResult, err error) (RepositoryResult, error) {
	result.Error = err.Error()
	return result, err
}

func observeMirror(repoPath, remoteName string, endpoint config.Endpoint, resolver transport.Resolver, git Git, result *MirrorResult) error {
	resolved, err := resolver.Resolve(endpoint)
	if err != nil {
		return fmt.Errorf("resolve mirror: %w", err)
	}
	if err := git.ConfigureRemote(repoPath, remoteName, resolved.URL); err != nil {
		return fmt.Errorf("configure mirror: %w", err)
	}
	if err := git.Fetch(repoPath, remoteName); err != nil {
		return fmt.Errorf("fetch mirror: %w", err)
	}
	if err := git.SetRemoteHead(repoPath, remoteName); err != nil {
		return fmt.Errorf("resolve mirror HEAD: %w", err)
	}
	counts, err := git.RevListLeftRightCount(repoPath, "canonical/HEAD", remoteName+"/HEAD")
	if err != nil {
		return fmt.Errorf("compare mirror: %w", err)
	}
	left, right, err := ParseRevListCount(counts)
	if err != nil {
		return fmt.Errorf("parse mirror divergence: %w", err)
	}
	interpreted := InterpretDivergence(left, right)
	commit, err := git.RevParseShort(repoPath, remoteName+"/HEAD")
	if err != nil {
		return fmt.Errorf("resolve mirror commit evidence: %w", err)
	}
	result.Commit = strings.TrimSpace(commit)
	result.State = interpreted.State
	result.Ahead = interpreted.Ahead
	result.Behind = interpreted.Behind
	return nil
}

func newMirrorResult(endpoint config.Endpoint) (MirrorResult, error) {
	provider := strings.TrimSpace(endpoint.Provider)
	path := strings.Trim(strings.TrimSpace(endpoint.Path), "/")
	if path == "" {
		var err error
		path, err = safeLegacyPath(endpoint.URL)
		if err != nil {
			return MirrorResult{Provider: provider, Ref: "HEAD", State: StateError}, fmt.Errorf("derive safe mirror identity: %w", err)
		}
	}
	return MirrorResult{
		Target:   provider + ":" + path,
		Provider: provider,
		Path:     path,
		Ref:      "HEAD",
		State:    StateError,
	}, nil
}

func safeLegacyPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	var path string
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("parse legacy URL")
		}
		path = parsed.Path
	} else if at := strings.LastIndex(raw, "@"); at >= 0 {
		if colon := strings.Index(raw[at+1:], ":"); colon >= 0 {
			path = raw[at+1+colon+1:]
		}
	}
	path = strings.Trim(strings.TrimSpace(path), "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("legacy URL does not contain a safe repository path")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `\\:@?#`) {
			return "", fmt.Errorf("legacy URL contains an unsafe repository path")
		}
	}
	return path, nil
}

func ParseRevListCount(output string) (int, int, error) {
	fields := strings.Fields(output)
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("expected two rev-list counts, got %q", strings.TrimSpace(output))
	}
	left, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse left rev-list count: %w", err)
	}
	right, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse right rev-list count: %w", err)
	}
	return left, right, nil
}

func InterpretDivergence(left, right int) Result {
	result := Result{Ahead: right, Behind: left}
	switch {
	case left == 0 && right == 0:
		result.State = StateEqual
	case left > 0 && right == 0:
		result.State = StateBehind
	case left == 0 && right > 0:
		result.State = StateAhead
	default:
		result.State = StateDiverged
	}
	return result
}
