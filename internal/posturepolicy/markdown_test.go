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

func TestRenderMarkdownEscapesNormalizedAndPolicyText(t *testing.T) {
	profile := Profile{
		Kind:    ProfileKind,
		Version: ProfileVersion,
		ID:      "baseline",
		Rules: []Rule{{
			ID:          "unsafe#rule",
			Area:        "docs\n## injected",
			Fact:        "fact`with`ticks",
			Operator:    OperatorEquals,
			Expected:    json.RawMessage(`true`),
			Severity:    SeverityHigh,
			Title:       "title\n## injected <script>",
			Remediation: []string{"fix\n- fake item"},
		}},
		Exceptions: []Exception{},
	}
	inputs := NewInputs("acme/project")
	inputs.Facts["fact`with`ticks"] = FactInput{
		State: posture.StateObserved,
		Value: json.RawMessage(`false`),
		Evidence: []posture.Evidence{{
			Source:    "source`value",
			Reference: "ref\n## injected",
			Detail:    "detail\n<a href='https://example.invalid'>link</a>",
		}},
	}
	report, err := Evaluate(profile, inputs, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	markdown := RenderMarkdown(report)
	for _, unsafe := range []string{"\n## injected", "<script>", "\n<a href=", "\n- fake item"} {
		if strings.Contains(markdown, unsafe) {
			t.Fatalf("markdown contains unescaped injected structure %q:\n%s", unsafe, markdown)
		}
	}
	if !strings.Contains(markdown, `docs\n\#\# injected`) {
		t.Fatalf("escaped area not rendered as literal text:\n%s", markdown)
	}
	if !strings.Contains(markdown, "``fact`with`ticks``") {
		t.Fatalf("fact containing backticks was not safely code-delimited:\n%s", markdown)
	}
}
