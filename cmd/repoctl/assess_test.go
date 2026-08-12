package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"repoctl/internal/assessment"
)

func TestAssessCreatesCanonicalValidatedSkeleton(t *testing.T) {
	path := filepath.Join(t.TempDir(), "assessment.json")
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"assess", path})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if got := stdout.String(); got != "created assessment template "+path+"\n" {
		t.Fatalf("stdout = %q", got)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report, err := assessment.Parse(data)
	if err != nil {
		t.Fatalf("generated report is invalid: %v", err)
	}
	if want := assessment.NewSkeleton(); !reflect.DeepEqual(report, want) {
		t.Fatalf("generated report does not match canonical skeleton\ngot: %#v\nwant: %#v", report, want)
	}
}

func TestAssessRefusesToOverwriteExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "assessment.json")
	const original = "preserve me\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := withStderr(t, &stderr, func() int {
		return run([]string{"assess", path})
	})
	if code != 1 {
		t.Fatalf("run returned %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "assessment report already exists") {
		t.Fatalf("stderr = %q, want already-exists error", stderr.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != original {
		t.Fatalf("existing file changed: got %q, want %q", got, original)
	}
}

func TestAssessDoesNotCreateParentDirectories(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "missing", "nested")
	path := filepath.Join(parent, "assessment.json")

	var stderr bytes.Buffer
	code := withStderr(t, &stderr, func() int {
		return run([]string{"assess", path})
	})
	if code != 1 {
		t.Fatalf("run returned %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "create assessment report") {
		t.Fatalf("stderr = %q, want create error", stderr.String())
	}
	if _, err := os.Stat(parent); !os.IsNotExist(err) {
		t.Fatalf("parent directory unexpectedly exists or stat error = %v", err)
	}
}

func TestAssessRequiresExactlyOneFile(t *testing.T) {
	var stderr bytes.Buffer
	code := withStderr(t, &stderr, func() int {
		return run([]string{"assess"})
	})
	if code != 1 {
		t.Fatalf("run returned %d, want 1", code)
	}
	if got := stderr.String(); got != "usage: repoctl assess FILE\n" {
		t.Fatalf("stderr = %q", got)
	}
}

func TestAssessHelp(t *testing.T) {
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"assess", "--help"})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if got := stdout.String(); got != "usage: repoctl assess FILE\n" {
		t.Fatalf("stdout = %q", got)
	}
}
