package posturepolicy

import (
	"encoding/json"
	"testing"
	"time"

	"repoctl/internal/posture"
)

func TestEqualsDoesNotRoundLargeJSONIntegers(t *testing.T) {
	profile := Profile{
		Kind:       ProfileKind,
		Version:    ProfileVersion,
		ID:         "baseline",
		Rules:      []Rule{{ID: "exact", Area: "repository", Fact: "count", Operator: OperatorEquals, Expected: json.RawMessage(`9007199254740992`), Severity: SeverityHigh, Title: "Exact count", Remediation: []string{}}},
		Exceptions: []Exception{},
	}
	inputs := NewInputs("acme/project")
	inputs.Facts["count"] = FactInput{State: posture.StateObserved, Value: json.RawMessage(`9007199254740993`), Evidence: []posture.Evidence{}}
	report, err := Evaluate(profile, inputs, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if got := report.Evaluations[0].Status; got != StatusFail {
		t.Fatalf("status = %q, want fail", got)
	}
}

func TestNumericOperatorsCompareDecimalsAndExponentsExactly(t *testing.T) {
	cases := []struct {
		name     string
		operator Operator
		observed string
		expected string
		want     ResultStatus
	}{
		{name: "equivalent decimal forms", operator: OperatorEquals, observed: `1.00e2`, expected: `100`, want: StatusPass},
		{name: "at least beyond float64 precision", operator: OperatorAtLeast, observed: `9007199254740993`, expected: `9007199254740992`, want: StatusPass},
		{name: "at most negative decimals", operator: OperatorAtMost, observed: `-1.0000000000000001`, expected: `-1`, want: StatusPass},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			profile := Profile{
				Kind:       ProfileKind,
				Version:    ProfileVersion,
				ID:         "baseline",
				Rules:      []Rule{{ID: "number", Area: "repository", Fact: "number", Operator: tc.operator, Expected: json.RawMessage(tc.expected), Severity: SeverityHigh, Title: "Number", Remediation: []string{}}},
				Exceptions: []Exception{},
			}
			inputs := NewInputs("acme/project")
			inputs.Facts["number"] = FactInput{State: posture.StateObserved, Value: json.RawMessage(tc.observed), Evidence: []posture.Evidence{}}
			report, err := Evaluate(profile, inputs, time.Unix(0, 0).UTC())
			if err != nil {
				t.Fatal(err)
			}
			if got := report.Evaluations[0].Status; got != tc.want {
				t.Fatalf("status = %q, want %q", got, tc.want)
			}
		})
	}
}
