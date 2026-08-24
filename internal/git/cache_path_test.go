package git

import (
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"
)

func TestMirrorPathUsesExplicitAbsoluteCacheRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "cache root")
	t.Setenv(cacheDirectoryEnvironment, root)

	got, err := MirrorPath("repo.operator-acceptance")
	if err != nil {
		t.Fatalf("MirrorPath() error = %v", err)
	}

	segment := "uid-" + base64.RawURLEncoding.EncodeToString([]byte("repo.operator-acceptance")) + ".git"
	want := filepath.Join(root, segment)
	if got != want {
		t.Fatalf("MirrorPath() = %q, want %q", got, want)
	}
}

func TestMirrorPathRejectsRelativeCacheRoot(t *testing.T) {
	t.Setenv(cacheDirectoryEnvironment, filepath.Join("relative", "cache"))

	_, err := MirrorPath("repo.operator-acceptance")
	if err == nil {
		t.Fatal("MirrorPath() error = nil, want relative cache root rejection")
	}
	if !strings.Contains(err.Error(), "REPORA_CACHE_DIR must be an absolute path") {
		t.Fatalf("MirrorPath() error = %q, want absolute-path diagnostic", err)
	}
}
