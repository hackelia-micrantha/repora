package assessment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommittedExampleCoversAssessmentAcceptanceConcepts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "repository-assessment-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	report, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse(example) error = %v", err)
	}

	presentTypes := map[string]bool{}
	for _, finding := range report.Findings {
		presentTypes[finding.Type] = true
	}
	for _, findingType := range []string{"question", "finding", "recommendation", "tradeoff", "risk", "gap"} {
		if !presentTypes[findingType] {
			t.Errorf("example is missing required finding type %q", findingType)
		}
	}

	strongEvidence := false
	for _, evidence := range report.Evidence {
		if evidence.Strength == "strong" {
			strongEvidence = true
			break
		}
	}
	if !strongEvidence {
		t.Fatal("example must demonstrate a classified strong evidence item")
	}
}
