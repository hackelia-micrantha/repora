package managedartifact

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/transport"
)

func TestPusherPushesPreparedCommitWithExactBaseLease(t *testing.T) {
	requireIntegration(t)
	fixture := newPushFixture(t)
	pusher := newPusher(gitwrap.Client{}, fixture.cache)

	results, err := pusher.Push(fixture.spec, fixture.plan, fixture.prepared, fixture.observer)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Pushed {
		t.Fatalf("results = %+v, want one successful push", results)
	}
	if got := gitOutput(t, fixture.remote, "rev-parse", "refs/heads/main"); got != fixture.prepared[0].CommitOID {
		t.Fatalf("remote main = %s, want candidate %s", got, fixture.prepared[0].CommitOID)
	}
	if got := gitOutput(t, fixture.remote, "rev-parse", "refs/heads/main^"); got != fixture.plan.Repositories[0].BaseOID {
		t.Fatalf("remote candidate parent = %s, want reviewed base %s", got, fixture.plan.Repositories[0].BaseOID)
	}
}

func TestPusherRejectsStaleRemoteBeforeMutation(t *testing.T) {
	requireIntegration(t)
	fixture := newPushFixture(t)
	advanceCanonical(t, fixture.work, fixture.remote, "concurrent before preflight")
	concurrentOID := gitOutput(t, fixture.remote, "rev-parse", "refs/heads/main")

	results, err := newPusher(gitwrap.Client{}, fixture.cache).Push(fixture.spec, fixture.plan, fixture.prepared, fixture.observer)
	if !errors.Is(err, ErrStale) {
		t.Fatalf("Push() error = %v, want ErrStale", err)
	}
	if len(results) != 0 {
		t.Fatalf("results = %+v, want no push attempts", results)
	}
	if got := gitOutput(t, fixture.remote, "rev-parse", "refs/heads/main"); got != concurrentOID {
		t.Fatalf("remote main = %s, want concurrent %s", got, concurrentOID)
	}
}

func TestPusherLeaseRejectsRaceAfterFreshPreflight(t *testing.T) {
	requireIntegration(t)
	fixture := newPushFixture(t)
	racing := &racingPushGit{Client: gitwrap.Client{}}
	racing.beforePush = func() {
		advanceCanonical(t, fixture.work, fixture.remote, "concurrent after preflight")
	}
	pusher := newPusher(racing, fixture.cache)

	results, err := pusher.Push(fixture.spec, fixture.plan, fixture.prepared, fixture.observer)
	if err == nil {
		t.Fatal("Push() error = nil, want lease rejection")
	}
	if len(results) != 1 || results[0].Pushed {
		t.Fatalf("results = %+v, want one failed push attempt", results)
	}
	concurrentOID := gitOutput(t, fixture.remote, "rev-parse", "refs/heads/main")
	if concurrentOID == fixture.prepared[0].CommitOID || concurrentOID == fixture.plan.Repositories[0].BaseOID {
		t.Fatalf("remote main = %s, want independently advanced commit", concurrentOID)
	}
}

type racingPushGit struct {
	gitwrap.Client
	beforePush func()
}

func (g *racingPushGit) ForcePushBranchWithLease(repoPath, remote, srcRef, dstBranch, expectedOldOID string) error {
	if g.beforePush != nil {
		before := g.beforePush
		g.beforePush = nil
		before()
	}
	return g.Client.ForcePushBranchWithLease(repoPath, remote, srcRef, dstBranch, expectedOldOID)
}

type pushFixture struct {
	work     string
	remote   string
	cache    func(string) (string, error)
	observer *GitREADMEObserver
	spec     config.Spec
	plan     Plan
	prepared []PreparedCommit
}

func newPushFixture(t *testing.T) pushFixture {
	t.Helper()
	work, remote := newLocalCanonical(t)
	if err := os.WriteFile(filepath.Join(work, READMEPath), []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "keep.txt"), []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "add", READMEPath, "keep.txt")
	runGit(t, work, "commit", "-m", "seed push fixture")
	runGit(t, work, "push", remote, "main")

	cachePath := filepath.Join(t.TempDir(), "cache.git")
	cache := func(string) (string, error) { return cachePath, nil }
	resolver := func(endpoint config.Endpoint) (transport.ResolvedRemote, error) {
		return transport.ResolvedRemote{Provider: endpoint.Provider, Path: endpoint.Path, URL: remote, Transport: transport.HTTPS}, nil
	}
	observer := newGitREADMEObserver(gitwrap.Client{}, resolver, cache)
	repo := observerTestRepo()
	repo.Mirrors = []config.Endpoint{{Provider: "github", Path: "example/demo"}}
	repo.Artifacts.Readme = &config.ReadmeArtifact{Template: "templates/README.md.tmpl"}
	spec := config.Spec{Repos: []config.Repo{repo}}

	observed, err := observer.ObserveREADME(repo)
	if err != nil {
		t.Fatal(err)
	}
	desired := "new\n"
	diff, err := ReviewDiff(true, observed.Content, []byte(desired))
	if err != nil {
		t.Fatal(err)
	}
	present := true
	plan := Plan{
		Kind:    PlanKind,
		Version: PlanVersion,
		Repositories: []RepositoryPlan{{
			UID:     repo.UID,
			ID:      repo.ID,
			Target:  Target{Provider: repo.Canonical.Provider, Path: repo.Canonical.Path, Branch: observed.Branch},
			BaseOID: observed.BaseOID,
			Actions: []Action{{
				Type:           ActionWriteREADME,
				Path:           READMEPath,
				Observed:       ObservedState{Present: &present, Mode: observed.Mode, SHA256: DigestSHA256(observed.Content)},
				Desired:        DesiredState{Mode: observed.Mode, SHA256: DigestSHA256([]byte(desired)), Content: &desired},
				TemplateSHA256: strings.Repeat("c", 64),
				Diff:           diff,
			}},
		}},
	}
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
	prepared, err := newCommitPreparer(gitwrap.Client{}, cache).Prepare(spec, plan, observer)
	if err != nil {
		t.Fatal(err)
	}
	return pushFixture{work: work, remote: remote, cache: cache, observer: observer, spec: spec, plan: plan, prepared: prepared}
}

func advanceCanonical(t *testing.T, work, remote, message string) {
	t.Helper()
	path := filepath.Join(work, "concurrent.txt")
	content := []byte(message + "\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "add", "concurrent.txt")
	runGit(t, work, "commit", "-m", message)
	runGit(t, work, "push", remote, "main")
}
