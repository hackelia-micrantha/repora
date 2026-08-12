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

	requiredTypes := map[string]bool{
		"question":       false,
		"finding":        false,
		"recommendation": false,
		"tradeoff":       false,
		"risk":           false,
		"gap":            false,
	}
	for _, finding := range report.Findings {
		if _, required := requiredTypes[finding.Type]; required {
			requiredTypes[finding.Type] = true
		}
	}
	for findingType, present := range requiredTypes {
		if !present {
			t.Errorf("example is missing required finding type %q", findingType)
		}
	}

	if len(report.Evidence) == 0 {
		t.Fatal("example must demonstrate evidence strength classification")
	}
	for _, evidence := range report.Evidence {
		if evidence.Strength != "" {
			return
		}
	}
	t.Fatal("example evidence has no strength classification")
}
