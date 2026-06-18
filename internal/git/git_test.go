package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSafePathSegmentEncodesAllIdentities(t *testing.T) {
	got, err := SafePathSegment("repo.org.payments-api")
	if err != nil {
		t.Fatalf("SafePathSegment returned error: %v", err)
	}
	if !strings.HasPrefix(got, "uid-") {
		t.Fatalf("segment = %q, want uid- prefix", got)
	}
	if strings.Contains(got, "/") {
		t.Fatalf("segment = %q, must not contain path separators", got)
	}
}

func TestSafePathSegmentAvoidsEncodedNamespaceCollision(t *testing.T) {
	unsafeSegment, err := SafePathSegment("repo/org")
	if err != nil {
		t.Fatalf("SafePathSegment returned error for unsafe identity: %v", err)
	}
	literalSegment, err := SafePathSegment("b64-cmVwby9vcmc")
	if err != nil {
		t.Fatalf("SafePathSegment returned error for literal identity: %v", err)
	}
	if unsafeSegment == literalSegment {
		t.Fatalf("segments collided: %q", unsafeSegment)
	}
}

func TestSafePathSegmentRejectsEmptyIdentity(t *testing.T) {
	if _, err := SafePathSegment("  "); err == nil {
		t.Fatal("SafePathSegment returned nil error, want empty identity rejection")
	}
}

func TestEnsureMirrorReclonesInvalidPath(t *testing.T) {
	repoDir := t.TempDir()
	sourceDir := filepath.Join(repoDir, "source.git")
	if err := os.Mkdir(sourceDir, 0o700); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := run("", "init", "--bare", sourceDir); err != nil {
		t.Fatalf("init bare source repo: %v", err)
	}

	cacheDir := filepath.Join(repoDir, "cache")
	if err := os.Mkdir(cacheDir, 0o700); err != nil {
		t.Fatalf("create cache dir: %v", err)
	}
	invalidPath := filepath.Join(cacheDir, "payments-api.git")
	if err := os.WriteFile(invalidPath, []byte("not a repo"), 0o600); err != nil {
		t.Fatalf("write invalid path: %v", err)
	}

	client := Client{}
	if err := client.EnsureMirror(invalidPath, sourceDir); err != nil {
		t.Fatalf("EnsureMirror returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(invalidPath, "HEAD")); err != nil {
		t.Fatalf("expected mirror repo HEAD file after reclone, got: %v", err)
	}
}

func TestEnsureMirrorKeepsValidMirror(t *testing.T) {
	repoDir := t.TempDir()
	sourceDir := filepath.Join(repoDir, "source.git")
	if err := os.Mkdir(sourceDir, 0o700); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := run("", "init", "--bare", sourceDir); err != nil {
		t.Fatalf("init bare source repo: %v", err)
	}

	mirrorDir := filepath.Join(repoDir, "mirror.git")
	client := Client{}
	if err := client.EnsureMirror(mirrorDir, sourceDir); err != nil {
		t.Fatalf("first EnsureMirror returned error: %v", err)
	}
	if err := client.EnsureMirror(mirrorDir, sourceDir); err != nil {
		t.Fatalf("second EnsureMirror returned error: %v", err)
	}
}

func TestSyncMirrorFromRemoteAndPushMirrorCopiesCanonicalRefs(t *testing.T) {
	repoDir := t.TempDir()
	workDir := filepath.Join(repoDir, "work")
	canonicalDir := filepath.Join(repoDir, "canonical.git")
	mirrorDir := filepath.Join(repoDir, "mirror.git")
	cacheDir := filepath.Join(repoDir, "cache.git")

	if err := run("", "init", workDir); err != nil {
		t.Fatalf("init work repo: %v", err)
	}
	if err := run(workDir, "config", "user.email", "repora@example.invalid"); err != nil {
		t.Fatalf("configure user email: %v", err)
	}
	if err := run(workDir, "config", "user.name", "Repora Test"); err != nil {
		t.Fatalf("configure user name: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "README.md"), []byte("initial\n"), 0o600); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	if err := run(workDir, "add", "README.md"); err != nil {
		t.Fatalf("add readme: %v", err)
	}
	if err := run(workDir, "commit", "-m", "initial"); err != nil {
		t.Fatalf("commit initial: %v", err)
	}
	if err := run(workDir, "branch", "-M", "main"); err != nil {
		t.Fatalf("rename branch: %v", err)
	}
	if err := run("", "init", "--bare", canonicalDir); err != nil {
		t.Fatalf("init canonical repo: %v", err)
	}
	if err := run("", "init", "--bare", mirrorDir); err != nil {
		t.Fatalf("init mirror repo: %v", err)
	}
	if err := run(workDir, "remote", "add", "origin", canonicalDir); err != nil {
		t.Fatalf("add origin: %v", err)
	}
	if err := run(workDir, "push", "origin", "main"); err != nil {
		t.Fatalf("push canonical: %v", err)
	}

	client := Client{}
	if err := client.EnsureMirror(cacheDir, canonicalDir); err != nil {
		t.Fatalf("EnsureMirror returned error: %v", err)
	}
	if err := client.ConfigureRemote(cacheDir, "canonical", canonicalDir); err != nil {
		t.Fatalf("ConfigureRemote canonical returned error: %v", err)
	}
	if err := client.ConfigureRemote(cacheDir, "mirror", mirrorDir); err != nil {
		t.Fatalf("ConfigureRemote mirror returned error: %v", err)
	}
	if err := client.SyncMirrorFromRemote(cacheDir, "canonical"); err != nil {
		t.Fatalf("SyncMirrorFromRemote returned error: %v", err)
	}
	if err := client.PushMirror(cacheDir, "mirror"); err != nil {
		t.Fatalf("PushMirror returned error: %v", err)
	}

	canonicalHead, err := output(canonicalDir, "rev-parse", "refs/heads/main")
	if err != nil {
		t.Fatalf("rev-parse canonical main: %v", err)
	}
	mirrorHead, err := output(mirrorDir, "rev-parse", "refs/heads/main")
	if err != nil {
		t.Fatalf("rev-parse mirror main: %v", err)
	}
	if strings.TrimSpace(mirrorHead) != strings.TrimSpace(canonicalHead) {
		t.Fatalf("mirror main = %q, want canonical %q", strings.TrimSpace(mirrorHead), strings.TrimSpace(canonicalHead))
	}
}

func TestFetchPrunesStaleRemoteRefs(t *testing.T) {
	repoDir := t.TempDir()
	workDir := filepath.Join(repoDir, "work")
	canonicalDir := filepath.Join(repoDir, "canonical.git")
	cacheDir := filepath.Join(repoDir, "cache.git")

	if err := run("", "init", workDir); err != nil {
		t.Fatalf("init work repo: %v", err)
	}
	if err := run(workDir, "config", "user.email", "repora@example.invalid"); err != nil {
		t.Fatalf("configure user email: %v", err)
	}
	if err := run(workDir, "config", "user.name", "Repora Test"); err != nil {
		t.Fatalf("configure user name: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "README.md"), []byte("initial\n"), 0o600); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	if err := run(workDir, "add", "README.md"); err != nil {
		t.Fatalf("add readme: %v", err)
	}
	if err := run(workDir, "commit", "-m", "initial"); err != nil {
		t.Fatalf("commit initial: %v", err)
	}
	if err := run(workDir, "branch", "-M", "main"); err != nil {
		t.Fatalf("rename branch: %v", err)
	}
	if err := run(workDir, "branch", "stale"); err != nil {
		t.Fatalf("create stale branch: %v", err)
	}
	if err := run("", "init", "--bare", canonicalDir); err != nil {
		t.Fatalf("init canonical repo: %v", err)
	}
	if err := run(workDir, "remote", "add", "origin", canonicalDir); err != nil {
		t.Fatalf("add origin: %v", err)
	}
	if err := run(workDir, "push", "origin", "main", "stale"); err != nil {
		t.Fatalf("push branches: %v", err)
	}

	client := Client{}
	if err := client.EnsureMirror(cacheDir, canonicalDir); err != nil {
		t.Fatalf("EnsureMirror returned error: %v", err)
	}
	if err := client.ConfigureRemote(cacheDir, "canonical", canonicalDir); err != nil {
		t.Fatalf("ConfigureRemote canonical returned error: %v", err)
	}
	if err := client.Fetch(cacheDir, "canonical"); err != nil {
		t.Fatalf("initial Fetch returned error: %v", err)
	}
	if _, err := output(cacheDir, "show-ref", "--verify", "refs/remotes/canonical/stale"); err != nil {
		t.Fatalf("expected stale remote ref before prune: %v", err)
	}

	if err := run(workDir, "push", "origin", "--delete", "stale"); err != nil {
		t.Fatalf("delete stale branch: %v", err)
	}
	if err := client.Fetch(cacheDir, "canonical"); err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}

	if _, err := output(cacheDir, "show-ref", "--verify", "refs/remotes/canonical/stale"); err == nil {
		t.Fatal("stale remote ref still exists after pruned fetch")
	}
}

func TestRunTimesOutGitCommand(t *testing.T) {
	binDir := t.TempDir()
	writeFakeGit(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	oldTimeout := gitTimeout
	gitTimeout = 25 * time.Millisecond
	t.Cleanup(func() { gitTimeout = oldTimeout })

	start := time.Now()
	err := run("", "status")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("run returned nil error, want timeout")
	}
	if elapsed > time.Duration(3.2 * float64(time.Second)) {
		t.Fatalf("run took %s, want command timeout before fake git exits", elapsed)
	}
}

func writeFakeGit(t *testing.T, binDir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		sourcePath := filepath.Join(binDir, "fake_git.go")
		source := []byte("package main\n\nimport \"time\"\n\nfunc main() {\n\ttime.Sleep(3 * time.Second)\n}\n")
		if err := os.WriteFile(sourcePath, source, 0o600); err != nil {
			t.Fatalf("write fake git source: %v", err)
		}
		cmd := exec.Command("go", "build", "-o", filepath.Join(binDir, "git.exe"), sourcePath)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build fake git: %v: %s", err, out)
		}
		return
	}

	path := filepath.Join(binDir, "git")
	data := []byte("#!/bin/sh\nsleep 3\n")
	if err := os.WriteFile(path, data, 0o700); err != nil {
		t.Fatalf("write fake git: %v", err)
	}
}
