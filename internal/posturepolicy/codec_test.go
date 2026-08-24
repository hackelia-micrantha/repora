package posturepolicy

import (
	"encoding/json"
	"testing"
	"time"

	"repoctl/internal/posture"
)

func TestPolicyContractsRoundTripStrictly(t *testing.T) {
	profile := Profile{
		Kind:       ProfileKind,
		Version:    ProfileVersion,
		ID:         "baseline",
		Rules:      []Rule{{ID: "protected", Area: "repository", Fact: "repository.default_branch_protected", Operator: OperatorEquals, Expected: json.RawMessage(`true`), Severity: SeverityHigh, Title: "Protected", Remediation: []string{"Enable protection."}}},
		Exceptions: []Exception{},
	}
	profileData, err := profile.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseProfile(profileData); err != nil {
		t.Fatal(err)
	}

	inputs := NewInputs("acme/project")
	inputs.Facts["repository.default_branch_protected"] = FactInput{State: posture.StateObserved, Value: json.RawMessage(`false`), Evidence: []posture.Evidence{{Source: "test", Reference: "branch"}}}
	inputData, err := inputs.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	parsedInputs, err := ParseInputs(inputData)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Evaluate(profile, parsedInputs, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := report.Marshal(); err != nil {
		t.Fatal(err)
	}
}

func TestPolicyInputsRejectUnknownFieldsAndInvalidFactShape(t *testing.T) {
	unknownField := []byte(`{"kind":"repora.posture-policy-inputs","version":1,"repository":"acme/project","facts":{},"extra":true}`)
	if _, err := ParseInputs(unknownField); err == nil {
		t.Fatal("expected unknown field to fail")
	}

	inputs := NewInputs("acme/project")
	inputs.Facts["bad"] = FactInput{State: posture.StateUnknown, Value: json.RawMessage(`true`), Evidence: []posture.Evidence{}}
	if err := inputs.Validate(); err == nil {
		t.Fatal("expected unknown fact carrying a value to fail")
	}
}

func TestInformationalMismatchIsWarning(t *testing.T) {
	profile := Profile{
		Kind:       ProfileKind,
		Version:    ProfileVersion,
		ID:         "baseline",
		Rules:      []Rule{{ID: "info", Area: "documentation", Fact: "f", Operator: OperatorEquals, Expected: json.RawMessage(`true`), Severity: SeverityInfo, Title: "Informational check", Remediation: []string{}}},
		Exceptions: []Exception{},
	}
	inputs := NewInputs("acme/project")
	inputs.Facts["f"] = FactInput{State: posture.StateObserved, Value: json.RawMessage(`false`), Evidence: []posture.Evidence{}}
	report, err := Evaluate(profile, inputs, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if got := report.Evaluations[0].Status; got != StatusWarning {
		t.Fatalf("status = %q, want warning", got)
	}
	if SummaryBySeverity(report)[SeverityInfo] != 1 {
		t.Fatal("warning was not counted in findings summary")
	}
}
