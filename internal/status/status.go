package status

import (
	"fmt"
	"strconv"
	"strings"

	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/transport"
)

type State string

const (
	StateEqual    State = "EQUAL"
	StateBehind   State = "BEHIND"
	StateAhead    State = "AHEAD"
	StateDiverged State = "DIVERGED"
)

type Result struct {
	ID        string `json:"id"`
	UID       string `json:"uid"`
	State     State  `json:"state"`
	Ahead     int    `json:"ahead"`
	Behind    int    `json:"behind"`
	Canonical string `json:"-"`
	Mirror    string `json:"-"`
}

type Git interface {
	EnsureMirror(path, canonicalURL string) error
	ConfigureRemote(repoPath, name, url string) error
	Fetch(repoPath, name string) error
	SetRemoteHead(repoPath, name string) error
	RevListLeftRightCount(repoPath, left, right string) (string, error)
	RevParseShort(repoPath, rev string) (string, error)
}

func Check(repo config.Repo, git Git) (Result, error) {
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
