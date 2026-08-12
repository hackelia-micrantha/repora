package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListFindingsPrintsReportOrder(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "repository-assessment-v1.json")
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"list-findings", path})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	want := "routing-determinism\tinformational\timplemented\tfinding\t\"Document routing has deterministic regression fixtures\"\n" +
		"assessment-automation-deferred\tlow\topen\tgap\t\"Assessment automation remains outside this schema slice\"\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestListFindingsAllowsEmptyFindings(t *testing.T) {
	path := filepath.Join("..", "..", "templates", "assessments", "repository-assessment-v1.json")
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"list-findings", path})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty output", stdout.String())
	}
}

func TestListFindingsRejectsInvalidReport(t *testing.T) {
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
		return run([]string{"list-findings", path})
	})
	if code != 1 {
		t.Fatalf("run returned %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "unknown field") {
		t.Fatalf("stderr = %q, want validation error", stderr.String())
	}
}

func TestListFindingsRequiresExactlyOneFile(t *testing.T) {
	var stderr bytes.Buffer
	code := withStderr(t, &stderr, func() int {
		return run([]string{"list-findings"})
	})
	if code != 1 {
		t.Fatalf("run returned %d, want 1", code)
	}
	if got := stderr.String(); got != "usage: repoctl list-findings FILE\n" {
		t.Fatalf("stderr = %q", got)
	}
}

func TestListFindingsHelp(t *testing.T) {
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"list-findings", "--help"})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if got := stdout.String(); got != "usage: repoctl list-findings FILE\n" {
		t.Fatalf("stdout = %q", got)
	}
}
