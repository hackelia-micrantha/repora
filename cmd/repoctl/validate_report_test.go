package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateReportDoesNotRequireRepositoryConfig(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "repository-assessment-v1.json")
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"validate-report", path})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "valid assessment repora-routing-example") {
		t.Fatalf("stdout = %q, want validation confirmation", stdout.String())
	}
}

func TestValidateReportRejectsInvalidReport(t *testing.T) {
	templatePath := filepath.Join("..", "..", "templates", "assessments", "repository-assessment-v1.json")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `"summary":`, `"unexpected": true, "summary":`, 1))
	path := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := withStderr(t, &stderr, func() int {
		return run([]string{"validate-report", path})
	})
	if code != 1 {
		t.Fatalf("run returned %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "unknown field") {
		t.Fatalf("stderr = %q, want unknown-field error", stderr.String())
	}
}

func TestValidateReportRequiresExactlyOneFile(t *testing.T) {
	var stderr bytes.Buffer
	code := withStderr(t, &stderr, func() int {
		return run([]string{"validate-report"})
	})
	if code != 1 {
		t.Fatalf("run returned %d, want 1", code)
	}
	if got := stderr.String(); got != "usage: repoctl validate-report FILE\n" {
		t.Fatalf("stderr = %q", got)
	}
}

func TestValidateReportHelp(t *testing.T) {
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"validate-report", "--help"})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if got := stdout.String(); got != "usage: repoctl validate-report FILE\n" {
		t.Fatalf("stdout = %q", got)
	}
}
