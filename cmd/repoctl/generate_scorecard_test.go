package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateScorecardPrintsValidatedDimensionsInReportOrder(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "repository-assessment-v1.json")
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"generate-scorecard", path})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	want := "architecture\t4\t[\"routing-fixtures\",\"ast-route-boundary\"]\t\"The example is limited to the routing subsystem and uses direct repository evidence; it is not a whole-project architecture score.\"\n" +
		"documentation\t4\t[\"routing-fixtures\",\"ast-route-boundary\"]\t\"Routing behavior is documented alongside the checked-in configuration and fixtures; this score is illustrative and scoped to routing.\"\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestGenerateScorecardPreservesZeroScore(t *testing.T) {
	path := filepath.Join("..", "..", "templates", "assessments", "repository-assessment-v1.json")
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"generate-scorecard", path})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	want := "documentation\t0\t[]\t\"TODO: replace this placeholder dimension or provide evidence-backed rationale.\"\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestGenerateScorecardRejectsInvalidReport(t *testing.T) {
	templatePath := filepath.Join("..", "..", "templates", "assessments", "repository-assessment-v1.json")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `"score": 0`, `"score": 6`, 1))
	path := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := withStderr(t, &stderr, func() int {
		return run([]string{"generate-scorecard", path})
	})
	if code != 1 {
		t.Fatalf("run returned %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "score must be an integer from 0 through 5") {
		t.Fatalf("stderr = %q, want score validation error", stderr.String())
	}
}

func TestGenerateScorecardRequiresExactlyOneFile(t *testing.T) {
	var stderr bytes.Buffer
	code := withStderr(t, &stderr, func() int {
		return run([]string{"generate-scorecard"})
	})
	if code != 1 {
		t.Fatalf("run returned %d, want 1", code)
	}
	if got := stderr.String(); got != "usage: repoctl generate-scorecard FILE\n" {
		t.Fatalf("stderr = %q", got)
	}
}

func TestGenerateScorecardHelp(t *testing.T) {
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"generate-scorecard", "--help"})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if got := stdout.String(); got != "usage: repoctl generate-scorecard FILE\n" {
		t.Fatalf("stdout = %q", got)
	}
}
