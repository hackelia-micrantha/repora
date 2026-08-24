package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"repoctl/internal/posture"
	"repoctl/internal/posturepolicy"
)

func TestPostureReportCommandRendersDeterministicMarkdownAndJSON(t *testing.T) {
	dir := t.TempDir()
	profile := posturepolicy.Profile{
		Kind:       posturepolicy.ProfileKind,
		Version:    posturepolicy.ProfileVersion,
		ID:         "baseline",
		Rules:      []posturepolicy.Rule{{ID: "protected", Area: "repository", Fact: "repository.default_branch_protected", Operator: posturepolicy.OperatorEquals, Expected: json.RawMessage(`true`), Severity: posturepolicy.SeverityHigh, Title: "Default branch is protected", Remediation: []string{"Enable branch protection."}}},
		Exceptions: []posturepolicy.Exception{},
	}
	profileData, err := profile.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	profilePath := filepath.Join(dir, "policy.json")
	if err := os.WriteFile(profilePath, profileData, 0o600); err != nil {
		t.Fatal(err)
	}
	inputs := posturepolicy.NewInputs("acme/project")
	inputs.Facts["repository.default_branch_protected"] = posturepolicy.FactInput{State: posture.StateObserved, Value: json.RawMessage(`false`), Evidence: []posture.Evidence{{Source: "github", Reference: "branch-protection"}}}
	inputData, err := inputs.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	factsPath := filepath.Join(dir, "facts.json")
	if err := os.WriteFile(factsPath, inputData, 0o600); err != nil {
		t.Fatal(err)
	}

	var markdown bytes.Buffer
	code := withStdout(t, &markdown, func() int {
		return run([]string{"posture", "report", "--profile", profilePath, "--facts", factsPath, "--as-of", "2026-08-23"})
	})
	if code != 0 {
		t.Fatalf("markdown report returned %d", code)
	}
	if !strings.Contains(markdown.String(), "As of: `2026-08-23`") || !strings.Contains(markdown.String(), "Status: **fail**") {
		t.Fatalf("unexpected markdown:\n%s", markdown.String())
	}

	var jsonOutput bytes.Buffer
	code = withStdout(t, &jsonOutput, func() int {
		return run([]string{"posture", "report", "--profile", profilePath, "--facts", factsPath, "--as-of", "2026-08-23", "--format", "json"})
	})
	if code != 0 {
		t.Fatalf("json report returned %d", code)
	}
	var report posturepolicy.Report
	if err := json.Unmarshal(jsonOutput.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Kind != posturepolicy.ReportKind || report.AsOf != "2026-08-23" || report.Evaluations[0].Status != posturepolicy.StatusFail {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestPostureReportRequiresExplicitAsOfAndHasHelp(t *testing.T) {
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"posture", "report", "--help"})
	})
	if code != 0 || !strings.Contains(stdout.String(), "--as-of YYYY-MM-DD") {
		t.Fatalf("help code=%d output=%q", code, stdout.String())
	}

	var stderr bytes.Buffer
	code = withStderr(t, &stderr, func() int {
		return run([]string{"posture", "report", "--profile", "p.json", "--facts", "f.json"})
	})
	if code != 1 || !strings.Contains(stderr.String(), "--as-of YYYY-MM-DD") {
		t.Fatalf("missing as-of code=%d output=%q", code, stderr.String())
	}
}
