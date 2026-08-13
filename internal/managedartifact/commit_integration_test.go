package managedartifact

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/transport"
)

func TestCommitPreparerCreatesOnlyUnreferencedREADMECommit(t *testing.T) {
	work, remote := newLocalCanonical(t)
	if err := os.WriteFile(filepath.Join(work, "keep.txt"), []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, READMEPath), []byte("old\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(work, READMEPath), 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "add", "keep.txt", READMEPath)
	runGit(t, work, "commit", "-m", "seed files")
	runGit(t, work, "push", remote, "main")

	cachePath := filepath.Join(t.TempDir(), "cache.git")
	resolver := func(endpoint config.Endpoint) (transport.ResolvedRemote, error) {
		return transport.ResolvedRemote{Provider: endpoint.Provider, Path: endpoint.Path, URL: remote, Transport: transport.HTTPS}, nil
	}
	cache := func(string) (string, error) { return cachePath, nil }
	observer := newGitREADMEObserver(gitwrap.Client{}, resolver, cache)
	preparer := newCommitPreparer(gitwrap.Client{}, cache)
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
			UID: repo.UID,
			ID:  repo.ID,
			Target: Target{Provider: repo.Canonical.Provider, Path: repo.Canonical.Path, Branch: observed.Branch},
			BaseOID: observed.BaseOID,
			Actions: []Action{{
				Type: ActionWriteREADME,
				Path: READMEPath,
				Observed: ObservedState{Present: &present, Mode: observed.Mode, SHA256: DigestSHA256(observed.Content)},
				Desired: DesiredState{Mode: observed.Mode, SHA256: DigestSHA256([]byte(desired)), Content: &desired},
				TemplateSHA256: strings.Repeat("c", 64),
				Diff: diff,
			}},
		}},
	}
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}

	refsBefore := gitOutput(t, cachePath, "for-each-ref", "--format=%(refname) %(objectname)")
	remoteBefore := gitOutput(t, remote, "rev-parse", "refs/heads/main")
	prepared, err := preparer.Prepare(spec, plan, observer)
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared) != 1 {
		t.Fatalf("prepared count = %d, want 1", len(prepared))
	}
	candidate := prepared[0]
	if candidate.BaseOID != observed.BaseOID || candidate.CommitOID == "" || candidate.TreeOID == "" {
		t.Fatalf("prepared = %+v", candidate)
	}

	if got := gitOutput(t, cachePath, "rev-parse", candidate.CommitOID+"^"); got != observed.BaseOID {
		t.Fatalf("candidate parent = %s, want %s", got, observed.BaseOID)
	}
	if got := gitOutput(t, cachePath, "diff-tree", "--no-commit-id", "--name-only", "-r", observed.BaseOID, candidate.CommitOID); got != READMEPath {
		t.Fatalf("changed paths = %q, want %q", got, READMEPath)
	}
	readme := gitBytes(t, cachePath, "show", candidate.CommitOID+":"+READMEPath)
	if !bytes.Equal(readme, []byte(desired)) {
		t.Fatalf("candidate README = %q, want %q", readme, desired)
	}
	if got := gitBytes(t, cachePath, "show", candidate.CommitOID+":keep.txt"); !bytes.Equal(got, []byte("keep\n")) {
		t.Fatalf("candidate keep.txt = %q", got)
	}
	if got := gitOutput(t, cachePath, "ls-tree", candidate.CommitOID, READMEPath); !strings.HasPrefix(got, "100755 blob ") {
		t.Fatalf("candidate README tree entry = %q, want mode 100755", got)
	}
	if got := gitOutput(t, cachePath, "show", "-s", "--format=%an|%ae|%cn|%ce|%s", candidate.CommitOID); got != "Repora|repora@localhost.invalid|Repora|repora@localhost.invalid|"+managedREADMECommitMessage {
		t.Fatalf("candidate metadata = %q", got)
	}

	refsAfter := gitOutput(t, cachePath, "for-each-ref", "--format=%(refname) %(objectname)")
	if refsAfter != refsBefore {
		t.Fatalf("local refs changed during preparation\nbefore:\n%s\nafter:\n%s", refsBefore, refsAfter)
	}
	if remoteAfter := gitOutput(t, remote, "rev-parse", "refs/heads/main"); remoteAfter != remoteBefore {
		t.Fatalf("remote main changed from %s to %s", remoteBefore, remoteAfter)
	}
}

func gitBytes(t *testing.T, repoPath string, args ...string) []byte {
	t.Helper()
	cmdArgs := append([]string{"-C", repoPath}, args...)
	cmd := execCommand(t, cmdArgs...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", cmdArgs, err)
	}
	return out
}

func execCommand(t *testing.T, args ...string) *os.ExecCmd {
	panic("replaced by review")
}
