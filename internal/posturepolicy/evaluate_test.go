package posturepolicy

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"repoctl/internal/posture"
)

func TestEvaluateCoversPassFailExceptionUnknownAndUnavailable(t *testing.T) {
	profile := Profile{
		Kind:    ProfileKind,
		Version: ProfileVersion,
		ID:      "baseline",
		Rules: []Rule{
			{ID: "protected", Area: "repository", Fact: "repository.default_branch_protected", Operator: OperatorEquals, Expected: json.RawMessage(`true`), Severity: SeverityHigh, Title: "Default branch is protected", Remediation: []string{"Enable branch protection."}},
			{ID: "reviews", Area: "repository", Fact: "repository.required_reviews", Operator: OperatorAtLeast, Expected: json.RawMessage(`1`), Severity: SeverityMedium, Title: "Reviews are required", Remediation: []string{"Require one approving review."}},
			{ID: "security-doc", Area: "documentation", Fact: "repository.security_md_present", Operator: OperatorEquals, Expected: json.RawMessage(`true`), Severity: SeverityLow, Title: "Security policy exists", Remediation: []string{"Add SECURITY.md."}},
			{ID: "mirror", Area: "mirrors", Fact: "mirrors.reconciliation_state", Operator: OperatorEquals, Expected: json.RawMessage(`"in-sync"`), Severity: SeverityMedium, Title: "Mirror is reconciled", Remediation: []string{"Review mirror drift."}},
			{ID: "checks", Area: "ci", Fact: "repository.required_status_checks", Operator: OperatorNonEmpty, Severity: SeverityMedium, Title: "Status checks are configured", Remediation: []string{"Configure required checks."}},
		},
		Exceptions: []Exception{{RuleID: "reviews", Reason: "bootstrap repository", Owner: "platform", Expires: "2026-09-01"}},
	}
	inputs := Inputs{
		Repository: "hackelia-micrantha/repora",
		Facts: map[string]FactInput{
			"repository.default_branch_protected": observed(true, "branch-protection"),
			"repository.required_reviews":         observed(0, "branch-protection"),
			"repository.security_md_present":      observed(false, "tree"),
			"mirrors.reconciliation_state":        {State: posture.StateUnavailable, Evidence: []posture.Evidence{{Source: "mirror", Reference: "origin"}}},
			"repository.required_status_checks":   {State: posture.StateUnknown, Evidence: []posture.Evidence{{Source: "github", Reference: "branch-protection"}}},
		},
	}

	report, err := Evaluate(profile, inputs, time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	statuses := map[string]ResultStatus{}
	for _, evaluation := range report.Evaluations {
		statuses[evaluation.RuleID] = evaluation.Status
	}
	if statuses["protected"] != StatusPass {
		t.Fatalf("protected = %s, want pass", statuses["protected"])
	}
	if statuses["reviews"] != StatusExcepted {
		t.Fatalf("reviews = %s, want excepted", statuses["reviews"])
	}
	if statuses["security-doc"] != StatusFail {
		t.Fatalf("security-doc = %s, want fail", statuses["security-doc"])
	}
	if statuses["mirror"] != StatusUnavailable {
		t.Fatalf("mirror = %s, want unavailable", statuses["mirror"])
	}
	if statuses["checks"] != StatusUnknown {
		t.Fatalf("checks = %s, want unknown", statuses["checks"])
	}
}

func TestExpiredExceptionRemainsFinding(t *testing.T) {
	profile := Profile{
		Kind: ProfileKind, Version: ProfileVersion, ID: "baseline",
		Rules:      []Rule{{ID: "protected", Area: "repository", Fact: "protected", Operator: OperatorEquals, Expected: json.RawMessage(`true`), Severity: SeverityHigh, Title: "Protected", Remediation: []string{}}},
		Exceptions: []Exception{{RuleID: "protected", Reason: "migration", Owner: "platform", Expires: "2026-08-01"}},
	}
	report, err := Evaluate(profile, Inputs{Repository: "o/r", Facts: map[string]FactInput{"protected": observed(false, "branch")}}, time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	got := report.Evaluations[0]
	if got.Status != StatusFail || got.ExceptionGap != "exception expired" {
		t.Fatalf("got status=%s gap=%q", got.Status, got.ExceptionGap)
	}
}

func TestRenderMarkdownIsDeterministicAndVisible(t *testing.T) {
	profile := Profile{
		Kind: ProfileKind, Version: ProfileVersion, ID: "baseline",
		Rules: []Rule{
			{ID: "z-rule", Area: "security", Fact: "z", Operator: OperatorEquals, Expected: json.RawMessage(`true`), Severity: SeverityHigh, Title: "Z", Remediation: []string{"Fix Z."}},
			{ID: "a-rule", Area: "documentation", Fact: "a", Operator: OperatorEquals, Expected: json.RawMessage(`true`), Severity: SeverityLow, Title: "A", Remediation: []string{"Fix A."}},
		},
		Exceptions: []Exception{},
	}
	inputs := Inputs{Repository: "o/r", Facts: map[string]FactInput{
		"z": {State: posture.StateUnavailable, Evidence: []posture.Evidence{}},
		"a": observed(false, "tree"),
	}}
	report, err := Evaluate(profile, inputs, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	first := RenderMarkdown(report)
	second := RenderMarkdown(report)
	if first != second {
		t.Fatal("markdown output is not deterministic")
	}
	if !strings.Contains(first, "Unknown or unavailable evidence") || !strings.Contains(first, "`z` is unavailable") {
		t.Fatalf("markdown hides unavailable evidence:\n%s", first)
	}
	if strings.Index(first, "### documentation") > strings.Index(first, "### security") {
		t.Fatal("areas are not sorted")
	}
}

func TestProfileRequiresCompleteExceptions(t *testing.T) {
	profile := Profile{
		Kind:       ProfileKind,
		Version:    ProfileVersion,
		ID:         "baseline",
		Rules:      []Rule{{ID: "r", Area: "repository", Fact: "f", Operator: OperatorEquals, Expected: json.RawMessage(`true`), Severity: SeverityLow, Title: "Rule", Remediation: []string{}}},
		Exceptions: []Exception{{RuleID: "r", Reason: "", Owner: "platform", Expires: "2026-09-01"}},
	}
	if err := profile.Validate(); err == nil {
		t.Fatal("expected incomplete exception to fail validation")
	}
}

func observed(value any, reference string) FactInput {
	raw, _ := json.Marshal(value)
	return FactInput{State: posture.StateObserved, Value: raw, Evidence: []posture.Evidence{{Source: "test", Reference: reference}}}
}
