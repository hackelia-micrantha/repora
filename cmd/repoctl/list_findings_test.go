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
	want := "routing-refinement-question\tinformational\taccepted\tquestion\t\"How should AST selectors compose with route-first retrieval?\"\n" +
		"routing-determinism\tinformational\timplemented\tfinding\t\"Document routing has deterministic regression fixtures\"\n" +
		"routing-monotonic-refinement\tmedium\timplemented\trecommendation\t\"Keep AST selectors as monotonic route refinements\"\n" +
		"deterministic-routing-tradeoff\tlow\taccepted\ttradeoff\t\"Deterministic routing trades flexibility for auditability\"\n" +
		"route-fixture-drift-risk\tmedium\topen\trisk\t\"Intentional route changes can drift from fixture expectations\"\n" +
		"assessment-automation-deferred-at-snapshot\tlow\topen\tgap\t\"Assessment automation was deferred at this snapshot\"\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestListFindingsJSONQuotesTitle(t *testing.T) {
	examplePath := filepath.Join("..", "..", "examples", "repository-assessment-v1.json")
	data, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(
		string(data),
		`"title": "Document routing has deterministic regression fixtures",`,
		`"title": "Line one\tline two\nquote \" and control \u0001",`,
		1,
	))
	path := filepath.Join(t.TempDir(), "escaped-title.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"list-findings", path})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	wantLine := "routing-determinism\tinformational\timplemented\tfinding\t\"Line one\\tline two\\nquote \\\" and control \\u0001\"\n"
	if !strings.Contains(stdout.String(), wantLine) {
		t.Fatalf("stdout = %q, want line %q", stdout.String(), wantLine)
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
