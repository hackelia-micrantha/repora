package posturepolicy

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"repoctl/internal/posture"
)

func TestRenderMarkdownPreservesWhitespaceInsideJSONStringValues(t *testing.T) {
	profile := Profile{
		Kind:       ProfileKind,
		Version:    ProfileVersion,
		ID:         "baseline",
		Rules:      []Rule{{ID: "message", Area: "documentation", Fact: "message", Operator: OperatorEquals, Expected: json.RawMessage(`"hello   world"`), Severity: SeverityLow, Title: "Message", Remediation: []string{}}},
		Exceptions: []Exception{},
	}
	inputs := NewInputs("acme/project")
	inputs.Facts["message"] = FactInput{State: posture.StateObserved, Value: json.RawMessage(`"hello   world"`), Evidence: []posture.Evidence{}}
	report, err := Evaluate(profile, inputs, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	markdown := RenderMarkdown(report)
	if !strings.Contains(markdown, "`\"hello   world\"`") {
		t.Fatalf("JSON string whitespace changed in markdown:\n%s", markdown)
	}
}
