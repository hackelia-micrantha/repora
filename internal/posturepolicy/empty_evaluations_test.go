package posturepolicy

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestZeroRuleProfilesProduceValidEmptyReports(t *testing.T) {
	tests := []struct {
		name           string
		profileVersion int
		reportVersion  int
	}{
		{name: "v1", profileVersion: ProfileVersion, reportVersion: ReportVersion},
		{name: "v2", profileVersion: ProfileVersionV2, reportVersion: ReportVersionV2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			profile := Profile{
				Kind:       ProfileKind,
				Version:    tc.profileVersion,
				ID:         "no-rules",
				Rules:      []Rule{},
				Exceptions: []Exception{},
			}
			inputs := NewInputs("acme/project")
			report, err := Evaluate(profile, inputs, time.Unix(0, 0).UTC())
			if err != nil {
				t.Fatal(err)
			}
			if report.Version != tc.reportVersion || report.Evaluations == nil || len(report.Evaluations) != 0 {
				t.Fatalf("unexpected zero-rule report: version=%d evaluations=%#v", report.Version, report.Evaluations)
			}

			data, err := report.Marshal()
			if err != nil {
				t.Fatalf("valid empty report could not be marshaled: %v", err)
			}
			if !bytes.Contains(data, []byte(`"evaluations": []`)) {
				t.Fatalf("empty evaluations must be an array: %s", data)
			}
			var roundTripped Report
			if err := json.Unmarshal(data, &roundTripped); err != nil {
				t.Fatal(err)
			}
			if roundTripped.Evaluations == nil || len(roundTripped.Evaluations) != 0 {
				t.Fatalf("round-trip lost the empty array: %#v", roundTripped.Evaluations)
			}
			if err := roundTripped.Validate(); err != nil {
				t.Fatalf("round-trip report did not validate: %v", err)
			}

			markdown := RenderMarkdown(report)
			for _, want := range []string{"## Findings summary", "- high: 0", "## Policy evaluation", "## Unknown or unavailable evidence", "None."} {
				if !strings.Contains(markdown, want) {
					t.Fatalf("empty report missing %q:\n%s", want, markdown)
				}
			}
			if strings.Contains(markdown, "\n### ") {
				t.Fatalf("zero-rule report invented policy evaluations:\n%s", markdown)
			}

			for _, raw := range [][]byte{
				bytes.Replace(data, []byte(`"evaluations": []`), []byte(`"evaluations": null`), 1),
				bytes.Replace(data, []byte(`  "evaluations": []\n`), nil, 1),
			} {
				var invalid Report
				if err := json.Unmarshal(raw, &invalid); err != nil {
					// An unparseable missing-field fixture is also invalid; this path must not weaken validation.
					continue
				}
				if _, err := invalid.Marshal(); err == nil {
					t.Fatalf("missing/null evaluations were accepted: %s", raw)
				}
			}
		})
	}
}

func TestSortedEvaluationsRetainsNonNilEmptyAndCopyOrdering(t *testing.T) {
	if got := sortedEvaluations(nil); got == nil || len(got) != 0 {
		t.Fatalf("nil input should sort into a non-nil empty slice: %#v", got)
	}
	input := []Evaluation{
		{Area: "security", RuleID: "z"},
		{Area: "documentation", RuleID: "b"},
		{Area: "documentation", RuleID: "a"},
	}
	sorted := sortedEvaluations(input)
	if sorted[0].RuleID != "a" || sorted[1].RuleID != "b" || sorted[2].RuleID != "z" {
		t.Fatalf("non-empty sort order changed: %#v", sorted)
	}
	if input[0].RuleID != "z" {
		t.Fatal("sorting modified the original evaluation slice")
	}
}
