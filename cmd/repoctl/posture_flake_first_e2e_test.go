package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"repoctl/internal/posturepolicy"
)

func TestMicranthaFlakeFirstConformanceFixtureMatrix(t *testing.T) {
	type fixtureExpectation struct {
		name              string
		wantStatus        posturepolicy.ResultStatus
		wantExternalInput bool
		wantRemediation   bool
	}

	cases := []fixtureExpectation{
		{name: "compliant", wantStatus: posturepolicy.StatusPass},
		{name: "violating", wantStatus: posturepolicy.StatusFail, wantRemediation: true},
		{name: "not-applicable", wantStatus: posturepolicy.StatusNotApplicable},
		{name: "platform-exception-evidence", wantStatus: posturepolicy.StatusPass, wantExternalInput: true},
		{name: "ambiguous", wantStatus: posturepolicy.StatusUnknown},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixturePath := filepath.Join("testdata", "flake-first", tc.name+".json")
			profilePath := filepath.Join("..", "..", "examples", "posture", "micrantha-flake-first-policy-v2.json")

			var facts bytes.Buffer
			if code := withStdout(t, &facts, func() int {
				return run([]string{"posture", "converge", "--ci-environment", fixturePath})
			}); code != 0 {
				t.Fatalf("posture converge returned %d", code)
			}

			inputs, err := posturepolicy.ParseInputs(facts.Bytes())
			if err != nil {
				t.Fatalf("parse converged facts: %v", err)
			}
			if inputs.Repository != "acme/fixture" {
				t.Fatalf("repository = %q, want acme/fixture", inputs.Repository)
			}

			externalClass, hasExternal := inputs.Facts["ci_environment.external_input.macos-xcode.class"]
			if tc.wantExternalInput {
				if !hasExternal || externalClass.State != "observed" || string(externalClass.Value) != `"platform"` {
					t.Fatalf("platform external-input evidence = %#v, present=%v", externalClass, hasExternal)
				}
			} else if hasExternal {
				t.Fatalf("unexpected platform external-input evidence: %#v", externalClass)
			}

			dir := t.TempDir()
			factsPath := filepath.Join(dir, "facts.json")
			if err := os.WriteFile(factsPath, facts.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}

			var reportJSON bytes.Buffer
			if code := withStdout(t, &reportJSON, func() int {
				return run([]string{
					"posture", "report",
					"--profile", profilePath,
					"--facts", factsPath,
					"--as-of", "2026-09-25",
					"--format", "json",
				})
			}); code != 0 {
				t.Fatalf("posture report returned %d", code)
			}

			var report posturepolicy.Report
			if err := json.Unmarshal(reportJSON.Bytes(), &report); err != nil {
				t.Fatalf("decode report: %v", err)
			}
			if err := report.Validate(); err != nil {
				t.Fatalf("validate report: %v", err)
			}
			if report.Version != posturepolicy.ReportVersionV2 {
				t.Fatalf("report version = %d, want %d", report.Version, posturepolicy.ReportVersionV2)
			}
			if len(report.Evaluations) != 4 {
				t.Fatalf("evaluation count = %d, want 4", len(report.Evaluations))
			}

			for _, evaluation := range report.Evaluations {
				if evaluation.Status != tc.wantStatus {
					t.Fatalf("%s status = %q, want %q", evaluation.RuleID, evaluation.Status, tc.wantStatus)
				}
				if evaluation.Exception != nil {
					t.Fatalf("%s unexpectedly received an automatic exception: %#v", evaluation.RuleID, evaluation.Exception)
				}
				if tc.name == "ambiguous" {
					if evaluation.Applicability == nil || evaluation.Applicability.Decision != posturepolicy.ApplicabilityUnresolved {
						t.Fatalf("%s applicability = %#v, want unresolved", evaluation.RuleID, evaluation.Applicability)
					}
					if len(evaluation.Observed) != 0 {
						t.Fatalf("%s evaluated target fact despite unresolved applicability: %s", evaluation.RuleID, evaluation.Observed)
					}
				}
				if tc.name == "not-applicable" {
					if evaluation.Applicability == nil || evaluation.Applicability.Decision != posturepolicy.ApplicabilityNotApplicable {
						t.Fatalf("%s applicability = %#v, want not-applicable", evaluation.RuleID, evaluation.Applicability)
					}
				}
			}

			if tc.wantRemediation {
				found := false
				for _, evaluation := range report.Evaluations {
					if len(evaluation.Remediation) > 0 {
						found = true
						break
					}
				}
				if !found {
					t.Fatal("violating report omitted remediation")
				}
			}

			var markdown bytes.Buffer
			if code := withStdout(t, &markdown, func() int {
				return run([]string{
					"posture", "report",
					"--profile", profilePath,
					"--facts", factsPath,
					"--as-of", "2026-09-25",
					"--format", "markdown",
				})
			}); code != 0 {
				t.Fatalf("posture report markdown returned %d", code)
			}
			if !bytes.Contains(markdown.Bytes(), []byte("# Repository posture report")) {
				t.Fatalf("markdown report missing heading:\n%s", markdown.Bytes())
			}
			if tc.name == "not-applicable" && !bytes.Contains(markdown.Bytes(), []byte("not\\-applicable")) {
				t.Fatalf("N/A report does not expose not-applicable status:\n%s", markdown.Bytes())
			}
			if tc.name == "ambiguous" && !bytes.Contains(markdown.Bytes(), []byte("is unresolved")) {
				t.Fatalf("ambiguous report hides unresolved applicability:\n%s", markdown.Bytes())
			}
		})
	}
}
