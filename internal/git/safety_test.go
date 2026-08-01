package git

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnsureMirrorSupportsSpacesAndUnicodePaths(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source repository.git")
	if err := run("", "init", "--bare", source); err != nil {
		t.Fatalf("init source: %v", err)
	}

	mirror := filepath.Join(root, "cache with spaces", "réplica-東京.git")
	if err := (Client{}).EnsureMirror(mirror, source); err != nil {
		t.Fatalf("EnsureMirror returned error: %v", err)
	}
	if valid, err := isValidMirror(mirror); err != nil || !valid {
		t.Fatalf("mirror valid = %v, error = %v", valid, err)
	}
}

func TestEnsureMirrorRejectsMalformedParentDeterministically(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(parent, []byte("file"), 0o600); err != nil {
		t.Fatalf("write parent file: %v", err)
	}
	mirror := filepath.Join(parent, "mirror.git")

	first := (Client{}).EnsureMirror(mirror, filepath.Join(root, "missing.git"))
	second := (Client{}).EnsureMirror(mirror, filepath.Join(root, "missing.git"))
	if first == nil || second == nil {
		t.Fatalf("errors = %v / %v, want deterministic filesystem failure", first, second)
	}
	if first.Error() != second.Error() || !strings.Contains(first.Error(), "mirror") {
		t.Fatalf("errors = %q / %q, want stable actionable diagnostic", first, second)
	}
}

func TestEnsureMirrorRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges not guaranteed on Windows CI")
	}
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "cache")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	mirror := filepath.Join(link, "escaped.git")
	err := (Client{}).EnsureMirror(mirror, filepath.Join(root, "missing.git"))
	if err == nil || !strings.Contains(err.Error(), "symlink component") {
		t.Fatalf("error = %v, want symlink rejection", err)
	}
	if _, statErr := os.Stat(filepath.Join(outside, "escaped.git")); !os.IsNotExist(statErr) {
		t.Fatalf("outside workspace mutated: %v", statErr)
	}
}

func TestEnsureMirrorCleansIncompleteCloneAfterFailure(t *testing.T) {
	binDir := t.TempDir()
	writeFailingCloneGit(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	mirror := filepath.Join(t.TempDir(), "incomplete.git")
	err := (Client{}).EnsureMirror(mirror, "https://user:secret@example.invalid/org/repo.git")
	if err == nil {
		t.Fatal("EnsureMirror returned nil error")
	}
	if _, statErr := os.Stat(mirror); !os.IsNotExist(statErr) {
		t.Fatalf("incomplete mirror remains after failure: %v", statErr)
	}
	if strings.Contains(err.Error(), "secret") || !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("error leaked credential or omitted redaction marker: %q", err)
	}
}

func TestGitErrorsRedactCredentialsFromArgumentsAndOutput(t *testing.T) {
	binDir := t.TempDir()
	writeCredentialEchoGit(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	secretURL := "https://alice:swordfish@example.invalid/org/repo.git?token=abc123"
	_, err := output("", "ls-remote", secretURL)
	if err == nil {
		t.Fatal("output returned nil error")
	}
	text := err.Error()
	for _, secret := range []string{"alice", "swordfish", "abc123"} {
		if strings.Contains(text, secret) {
			t.Fatalf("error leaked %q: %q", secret, text)
		}
	}
	if !strings.Contains(text, "[REDACTED]") {
		t.Fatalf("error = %q, want redaction marker", text)
	}
}

func writeFailingCloneGit(t *testing.T, binDir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		writeWindowsFakeGit(t, binDir, `package main
import ("fmt"; "os")
func main(){ if len(os.Args)>4 { _=os.MkdirAll(os.Args[4],0700) }; fmt.Fprintln(os.Stderr,"https://user:secret@example.invalid/failure"); os.Exit(1) }
`)
		return
	}
	path := filepath.Join(binDir, "git")
	data := []byte("#!/bin/sh\nmkdir -p \"$4\"\necho 'https://user:secret@example.invalid/failure' >&2\nexit 1\n")
	if err := os.WriteFile(path, data, 0o700); err != nil {
		t.Fatalf("write fake git: %v", err)
	}
}

func writeCredentialEchoGit(t *testing.T, binDir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		writeWindowsFakeGit(t, binDir, `package main
import ("fmt"; "os")
func main(){ fmt.Fprintln(os.Stderr,"remote failed:",os.Args[len(os.Args)-1]); os.Exit(1) }
`)
		return
	}
	path := filepath.Join(binDir, "git")
	data := []byte("#!/bin/sh\necho \"remote failed: $2\" >&2\nexit 1\n")
	if err := os.WriteFile(path, data, 0o700); err != nil {
		t.Fatalf("write fake git: %v", err)
	}
}

func writeWindowsFakeGit(t *testing.T, binDir, source string) {
	t.Helper()
	sourcePath := filepath.Join(binDir, "fake_git.go")
	if err := os.WriteFile(sourcePath, []byte(source), 0o600); err != nil {
		t.Fatalf("write fake git source: %v", err)
	}
	cmd := execCommand("go", "build", "-o", filepath.Join(binDir, "git.exe"), sourcePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fake git: %v: %s", err, out)
	}
}
