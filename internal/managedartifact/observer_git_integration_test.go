package managedartifact

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/transport"
)

func TestGitREADMEObserverRefreshesCanonicalDefaultBranchExactly(t *testing.T) {
	requireIntegration(t)
	work, remote := newLocalCanonical(t)
	observer := localGitREADMEObserver(t, remote)
	repo := observerTestRepo()

	first, err := observer.ObserveREADME(repo)
	if err != nil {
		t.Fatal(err)
	}
	if first.Branch != "main" {
		t.Fatalf("branch = %q, want main", first.Branch)
	}
	if first.BaseOID != gitOutput(t, work, "rev-parse", "HEAD") {
		t.Fatalf("base OID = %q, want current canonical HEAD", first.BaseOID)
	}
	if first.Present || first.Mode != "" || len(first.Content) != 0 {
		t.Fatalf("initial observation = %+v, want absent README", first)
	}

	content := []byte("# Demo\r\n\r\nExact bytes.\r\n")
	readmePath := filepath.Join(work, READMEPath)
	if err := os.WriteFile(readmePath, content, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(readmePath, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "add", READMEPath)
	runGit(t, work, "commit", "-m", "add readme")
	runGit(t, work, "push", remote, "main")

	second, err := observer.ObserveREADME(repo)
	if err != nil {
		t.Fatal(err)
	}
	wantOID := gitOutput(t, work, "rev-parse", "HEAD")
	if second.BaseOID != wantOID || second.BaseOID == first.BaseOID {
		t.Fatalf("refreshed base OID = %q, first = %q, want %q", second.BaseOID, first.BaseOID, wantOID)
	}
	if second.Branch != "main" || !second.Present || second.Mode != "100755" {
		t.Fatalf("refreshed observation = %+v, want present executable README on main", second)
	}
	if !bytes.Equal(second.Content, content) {
		t.Fatalf("content = %q, want exact bytes %q", second.Content, content)
	}
}

func TestGitREADMEObserverRejectsOversizedBlobBeforeRead(t *testing.T) {
	requireIntegration(t)
	work, remote := newLocalCanonical(t)
	content := bytes.Repeat([]byte("x"), MaxTextBytes+1)
	if err := os.WriteFile(filepath.Join(work, READMEPath), content, 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "add", READMEPath)
	runGit(t, work, "commit", "-m", "add oversized readme")
	runGit(t, work, "push", remote, "main")

	_, err := localGitREADMEObserver(t, remote).ObserveREADME(observerTestRepo())
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("exceeds %d-byte limit", MaxTextBytes)) {
		t.Fatalf("ObserveREADME() error = %v, want bounded blob rejection", err)
	}
}

func TestGitREADMEObserverRejectsSymlinkREADME(t *testing.T) {
	requireIntegration(t)
	if runtime.GOOS == "windows" {
		t.Skip("symlink Git-mode assertion is Unix-specific")
	}
	work, remote := newLocalCanonical(t)
	if err := os.WriteFile(filepath.Join(work, "target.md"), []byte("target\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target.md", filepath.Join(work, READMEPath)); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "add", READMEPath, "target.md")
	runGit(t, work, "commit", "-m", "add symlink readme")
	runGit(t, work, "push", remote, "main")

	_, err := localGitREADMEObserver(t, remote).ObserveREADME(observerTestRepo())
	if err == nil || !strings.Contains(err.Error(), "regular Git blob with mode 100644 or 100755") {
		t.Fatalf("ObserveREADME() error = %v, want symlink mode rejection", err)
	}
}

func observerTestRepo() config.Repo {
	return config.Repo{
		ID:  "demo",
		UID: "repo.demo",
		Canonical: config.Endpoint{
			Provider: "gitlab",
			Path:     "example/demo",
		},
	}
}

func localGitREADMEObserver(t *testing.T, remote string) *GitREADMEObserver {
	t.Helper()
	cachePath := filepath.Join(t.TempDir(), "cache.git")
	return newGitREADMEObserver(
		gitwrap.Client{},
		func(endpoint config.Endpoint) (transport.ResolvedRemote, error) {
			return transport.ResolvedRemote{
				Provider:  endpoint.Provider,
				Path:      endpoint.Path,
				URL:       remote,
				Transport: transport.HTTPS,
			}, nil
		},
		func(string) (string, error) { return cachePath, nil },
	)
}

func newLocalCanonical(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	work := filepath.Join(root, "work")
	remote := filepath.Join(root, "canonical.git")
	runGit(t, "", "init", "-b", "main", work)
	runGit(t, work, "config", "user.name", "Repora Test")
	runGit(t, work, "config", "user.email", "repora@example.invalid")
	runGit(t, work, "commit", "--allow-empty", "-m", "initial")
	runGit(t, "", "clone", "--bare", work, remote)
	return work, remote
}

func runGit(t *testing.T, repoPath string, args ...string) {
	t.Helper()
	cmdArgs := append([]string(nil), args...)
	if repoPath != "" {
		cmdArgs = append([]string{"-C", repoPath}, cmdArgs...)
	}
	cmd := exec.Command("git", cmdArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", cmdArgs, err, out)
	}
}

func gitOutput(t *testing.T, repoPath string, args ...string) string {
	t.Helper()
	cmdArgs := append([]string(nil), args...)
	if repoPath != "" {
		cmdArgs = append([]string{"-C", repoPath}, cmdArgs...)
	}
	cmd := exec.Command("git", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", cmdArgs, err, out)
	}
	return strings.TrimSpace(string(out))
}
