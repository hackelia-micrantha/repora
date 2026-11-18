package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"repoctl/internal/posturepolicy"
)

func TestPostureReportCommandZeroRuleProfilesV1AndV2(t *testing.T) {
	for _, tc := range []struct {
		name           string
		profileVersion int
		reportVersion  int
	}{
		{name: "v1", profileVersion: posturepolicy.ProfileVersion, reportVersion: posturepolicy.ReportVersion},
		{name: "v2", profileVersion: posturepolicy.ProfileVersionV2, reportVersion: posturepolicy.ReportVersionV2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			profile := posturepolicy.Profile{
				Kind:       posturepolicy.ProfileKind,
				Version:    tc.profileVersion,
				ID:         "no-rules",
				Rules:      []posturepolicy.Rule{},
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
			factsData, err := posturepolicy.NewInputs("acme/project").Marshal()
			if err != nil {
				t.Fatal(err)
			}
			factsPath := filepath.Join(dir, "facts.json")
			if err := os.WriteFile(factsPath, factsData, 0o600); err != nil {
				t.Fatal(err)
			}

			args := []string{"posture", "report", "--profile", profilePath, "--facts", factsPath, "--as-of", "2026-10-22"}
			var output bytes.Buffer
			if code := withStdout(t, &output, func() int {
				return run(append(append([]string{}, args...), "--format", "json"))
			}); code != 0 {
				t.Fatalf("JSON report returned %d", code)
			}
			if !bytes.Contains(output.Bytes(), []byte(`"evaluations": []`)) {
				t.Fatalf("CLI did not render valid empty evaluations: %s", output.Bytes())
			}
			var report posturepolicy.Report
			if err := json.Unmarshal(output.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			if report.Version != tc.reportVersion || report.Evaluations == nil || len(report.Evaluations) != 0 {
				t.Fatalf("unexpected CLI report: %#v", report)
			}

			output.Reset()
			if code := withStdout(t, &output, func() int { return run(args) }); code != 0 {
				t.Fatalf("Markdown report returned %d", code)
			}
			if !strings.Contains(output.String(), "## Policy evaluation") ||
				!strings.Contains(output.String(), "- high: 0") ||
				!strings.Contains(output.String(), "## Unknown or unavailable evidence") {
				t.Fatalf("invalid Markdown for empty policy:\n%s", output.String())
			}
		})
	}
}
