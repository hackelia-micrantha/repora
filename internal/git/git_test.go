package git

import (
	"os"
	"path/filepath"
	"testing"
)

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
