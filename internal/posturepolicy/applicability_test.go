package posturepolicy

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"repoctl/internal/posture"
)

func conditionalPolicyProfile() Profile {
	return Profile{
		Kind:    ProfileKind,
		Version: ProfileVersionV2,
		ID:      "conditional-ci",
		Rules: []Rule{{
			ID:       "flake-required",
			Area:     "ci-environment",
			Fact:     "ci_environment.flake_present",
			Operator: OperatorEquals,
			Expected: json.RawMessage(`true`),
			Applicability: &RuleApplicability{
				Fact: "ci_environment.declared_applicability",
				ApplicableWhen: Condition{
					Operator: OperatorEquals,
					Expected: json.RawMessage(`"applicable"`),
				},
				NotApplicableWhen: Condition{
					Operator: OperatorEquals,
					Expected: json.RawMessage(`"not-applicable"`),
				},
			},
			Severity:    SeverityHigh,
			Title:       "Applicable CI uses a repository flake",
			Remediation: []string{"Move project CI tooling into the repository flake."},
		}},
		Exceptions: []Exception{},
	}
}

func TestConditionalApplicabilityStates(t *testing.T) {
	tests := []struct {
		name         string
		applicability *FactInput
		wantStatus   ResultStatus
		wantDecision ApplicabilityDecision
		target       *FactInput
	}{
		{
			name: "applicable evaluates target",
			applicability: factPtr(observed("applicable", "ci-profile")),
			target: factPtr(observed(false, "tree")),
			wantStatus: StatusFail,
			wantDecision: ApplicabilityApplicable,
		},
		{
			name: "explicit not applicable",
			applicability: factPtr(observed("not-applicable", "ci-profile")),
			wantStatus: StatusNotApplicable,
			wantDecision: ApplicabilityNotApplicable,
		},
		{
			name: "observed unresolved",
			applicability: factPtr(observed("unresolved", "ci-profile")),
			wantStatus: StatusUnknown,
			wantDecision: ApplicabilityUnresolved,
		},
		{
			name: "unknown applicability",
			applicability: &FactInput{State: posture.StateUnknown, Evidence: []posture.Evidence{{Source: "test", Reference: "ci-profile"}}},
			wantStatus: StatusUnknown,
			wantDecision: ApplicabilityUnknown,
		},
		{
			name: "unavailable applicability",
			applicability: &FactInput{State: posture.StateUnavailable, Evidence: []posture.Evidence{{Source: "test", Reference: "ci-profile"}}},
			wantStatus: StatusUnavailable,
			wantDecision: ApplicabilityUnavailable,
		},
		{
			name: "missing applicability",
			wantStatus: StatusUnknown,
			wantDecision: ApplicabilityUnknown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inputs := NewInputs("acme/project")
			if tc.applicability != nil {
				inputs.Facts["ci_environment.declared_applicability"] = *tc.applicability
			}
			if tc.target != nil {
				inputs.Facts["ci_environment.flake_present"] = *tc.target
			}
			report, err := Evaluate(conditionalPolicyProfile(), inputs, time.Unix(0, 0).UTC())
			if err != nil {
				t.Fatal(err)
			}
			if report.Version != ReportVersionV2 {
				t.Fatalf("report version = %d, want %d", report.Version, ReportVersionV2)
			}
			evaluation := report.Evaluations[0]
			if evaluation.Status != tc.wantStatus {
				t.Fatalf("status = %q, want %q", evaluation.Status, tc.wantStatus)
			}
			if evaluation.Applicability == nil || evaluation.Applicability.Decision != tc.wantDecision {
				t.Fatalf("applicability = %#v, want decision %q", evaluation.Applicability, tc.wantDecision)
			}
			if tc.wantDecision != ApplicabilityApplicable && len(evaluation.Observed) != 0 {
				t.Fatalf("target fact was unexpectedly evaluated: observed=%s", evaluation.Observed)
			}
		})
	}
}

func TestConditionalApplicabilityOverlappingConditionsFailClosed(t *testing.T) {
	profile := conditionalPolicyProfile()
	profile.Rules[0].Applicability.ApplicableWhen = Condition{Operator: OperatorNonEmpty}
	profile.Rules[0].Applicability.NotApplicableWhen = Condition{Operator: OperatorNonEmpty}
	inputs := NewInputs("acme/project")
	inputs.Facts["ci_environment.declared_applicability"] = observed("applicable", "ci-profile")

	_, err := Evaluate(profile, inputs, time.Unix(0, 0).UTC())
	if err == nil || !strings.Contains(err.Error(), "matches both") {
		t.Fatalf("overlapping applicability conditions error = %v", err)
	}
}

func TestConditionalApplicabilityPreservesExceptionBehaviorWhenApplicable(t *testing.T) {
	profile := conditionalPolicyProfile()
	profile.Exceptions = []Exception{{RuleID: "flake-required", Reason: "migration", Owner: "platform", Expires: "2026-09-01"}}
	inputs := NewInputs("acme/project")
	inputs.Facts["ci_environment.declared_applicability"] = observed("applicable", "ci-profile")
	inputs.Facts["ci_environment.flake_present"] = observed(false, "tree")

	report, err := Evaluate(profile, inputs, time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if report.Evaluations[0].Status != StatusExcepted {
		t.Fatalf("status = %q, want excepted", report.Evaluations[0].Status)
	}
	if report.Evaluations[0].Applicability == nil || report.Evaluations[0].Applicability.Decision != ApplicabilityApplicable {
		t.Fatalf("applicability lost through exception: %#v", report.Evaluations[0].Applicability)
	}
}

func TestV1RejectsConditionalApplicabilityAndRemainsV1(t *testing.T) {
	profile := conditionalPolicyProfile()
	profile.Version = ProfileVersion
	if err := profile.Validate(); err == nil {
		t.Fatal("v1 profile accepted v2 applicability")
	}

	v1 := Profile{
		Kind: ProfileKind, Version: ProfileVersion, ID: "v1",
		Rules: []Rule{{ID: "r", Area: "repository", Fact: "f", Operator: OperatorEquals, Expected: json.RawMessage(`true`), Severity: SeverityLow, Title: "v1", Remediation: []string{}}},
		Exceptions: []Exception{},
	}
	inputs := NewInputs("acme/project")
	inputs.Facts["f"] = observed(true, "test")
	report, err := Evaluate(v1, inputs, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if report.Version != ReportVersion || report.Evaluations[0].Applicability != nil {
		t.Fatalf("v1 report changed shape: %#v", report)
	}
}

func TestV2PolicyAndReportRoundTripStrictly(t *testing.T) {
	profile := conditionalPolicyProfile()
	data, err := profile.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseProfile(data)
	if err != nil {
		t.Fatal(err)
	}
	inputs := NewInputs("acme/project")
	inputs.Facts["ci_environment.declared_applicability"] = observed("not-applicable", "ci-profile")
	report, err := Evaluate(parsed, inputs, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := report.Marshal(); err != nil {
		t.Fatal(err)
	}
	if SummaryBySeverity(report)[SeverityHigh] != 0 {
		t.Fatal("not-applicable rule was counted as a finding")
	}
}

func TestConditionalApplicabilityMarkdownExplainsNAAAndUnresolvedEvidence(t *testing.T) {
	profile := conditionalPolicyProfile()
	inputs := NewInputs("acme/project")
	inputs.Facts["ci_environment.declared_applicability"] = observed("not-applicable", "ci-profile")
	report, err := Evaluate(profile, inputs, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	markdown := RenderMarkdown(report)
	for _, want := range []string{
		"Status: **not\\-applicable**",
		"Applicability decision: **not\\-applicable**",
		"Applicability fact: `ci_environment.declared_applicability`",
		"Applicability observed: `\"not-applicable\"`",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown missing %q:\n%s", want, markdown)
		}
	}
	if strings.Contains(markdown, "Remediation options:") {
		t.Fatalf("N/A markdown showed target remediation:\n%s", markdown)
	}

	inputs.Facts["ci_environment.declared_applicability"] = observed("unresolved", "ci-profile")
	report, err = Evaluate(profile, inputs, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	markdown = RenderMarkdown(report)
	if !strings.Contains(markdown, "applicability `ci_environment.declared_applicability` is unresolved") {
		t.Fatalf("unresolved applicability is hidden:\n%s", markdown)
	}
}

func factPtr(value FactInput) *FactInput {
	return &value
}
