package apply_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"repoctl/internal/apply"
	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/status"
)

func TestExecuteSynchronizesBehindMirrorUsingLocalGitRepos(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for integration test")
	}

	root := t.TempDir()
	setHomeEnv(t, root)

	canonicalBare := filepath.Join(root, "canonical.git")
	mirrorBare := filepath.Join(root, "mirror.git")
	canonicalWork := filepath.Join(root, "canonical-work")
	mirrorWork := filepath.Join(root, "mirror-work")

	git(t, root, "init", "--bare", canonicalBare)
	git(t, root, "init", "--bare", mirrorBare)
	git(t, canonicalBare, "symbolic-ref", "HEAD", "refs/heads/main")
	git(t, mirrorBare, "symbolic-ref", "HEAD", "refs/heads/main")

	git(t, root, "clone", canonicalBare, canonicalWork)
	git(t, canonicalWork, "config", "user.name", "Repora Test")
	git(t, canonicalWork, "config", "user.email", "repora@example.com")
	writeFile(t, filepath.Join(canonicalWork, "README.md"), "v1\n")
	git(t, canonicalWork, "add", "README.md")
	git(t, canonicalWork, "commit", "-m", "initial")
	git(t, canonicalWork, "branch", "-M", "main")
	git(t, canonicalWork, "push", "origin", "main")

	git(t, root, "clone", canonicalBare, mirrorWork)
	git(t, mirrorWork, "remote", "add", "mirror", mirrorBare)
	git(t, mirrorWork, "push", "mirror", "main")
	git(t, root, "-C", mirrorBare, "symbolic-ref", "HEAD", "refs/heads/main")

	writeFile(t, filepath.Join(canonicalWork, "README.md"), "v2\n")
	git(t, canonicalWork, "add", "README.md")
	git(t, canonicalWork, "commit", "-m", "canonical ahead")
	git(t, canonicalWork, "push", "origin", "main")

	repo := config.Repo{
		ID: "payments-api",
		Canonical: config.Endpoint{Provider: "gitlab", URL: canonicalBare},
		Mirrors: []config.Endpoint{{Provider: "github", URL: mirrorBare}},
		Mode: "mirror",
	}

	st, err := status.Check(repo, gitwrap.Client{})
	if err != nil {
		t.Fatalf("status check before apply: %v", err)
	}
	if st.State != status.StateBehind {
		t.Fatalf("state before apply = %s, want %s", st.State, status.StateBehind)
	}

	result, err := apply.Execute(repo, st, gitwrap.Client{}, false, false)
	if err != nil {
		t.Fatalf("apply execute: %v", err)
	}
	if !result.Applied {
		t.Fatalf("apply result Applied = false, want true")
	}
	if len(result.Actions) != 1 {
		t.Fatalf("action count = %d, want 1", len(result.Actions))
	}

	after, err := status.Check(repo, gitwrap.Client{})
	if err != nil {
		t.Fatalf("status check after apply: %v", err)
	}
	if after.State != status.StateEqual {
		t.Fatalf("state after apply = %s, want %s", after.State, status.StateEqual)
	}
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return string(out)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func setHomeEnv(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}
}
