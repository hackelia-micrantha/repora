//go:build !windows

package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureMirrorReportsReadOnlyWorkspaceFailure(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.git")
	if err := run("", "init", "--bare", source); err != nil {
		t.Fatalf("init source: %v", err)
	}

	workspace := filepath.Join(root, "read-only")
	if err := os.Mkdir(workspace, 0o700); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if err := os.Chmod(workspace, 0o500); err != nil {
		t.Fatalf("make workspace read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(workspace, 0o700) })

	mirror := filepath.Join(workspace, "mirror.git")
	err := (Client{}).EnsureMirror(mirror, source)
	if err == nil {
		_ = os.RemoveAll(mirror)
		t.Skip("test user can write through read-only permissions")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "permission") && !strings.Contains(strings.ToLower(err.Error()), "read-only") {
		t.Fatalf("error = %q, want actionable read-only filesystem diagnostic", err)
	}
	if _, statErr := os.Stat(mirror); !os.IsNotExist(statErr) {
		t.Fatalf("mirror remains after read-only failure: %v", statErr)
	}
}
