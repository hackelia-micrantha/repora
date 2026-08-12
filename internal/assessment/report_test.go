package assessment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCommittedExample(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "repository-assessment-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(data); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
}

func TestParseRejectsUnknownField(t *testing.T) {
	data := validReportJSON(t)
	data = []byte(strings.Replace(string(data), `"summary":`, `"unexpected": true, "summary":`, 1))
	if _, err := Parse(data); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Parse() error = %v, want unknown-field rejection", err)
	}
}

func TestParseRejectsUnknownEvidenceLink(t *testing.T) {
	data := validReportJSON(t)
	data = []byte(strings.Replace(string(data), `"evidence_ids": []`, `"evidence_ids": ["missing"]`, 1))
	if _, err := Parse(data); err == nil || !strings.Contains(err.Error(), "unknown evidence id") {
		t.Fatalf("Parse() error = %v, want unknown evidence rejection", err)
	}
}

func TestParseRejectsMissingDirtyField(t *testing.T) {
	data := validReportJSON(t)
	data = []byte(strings.Replace(string(data), `, "dirty": false`, "", 1))
	if _, err := Parse(data); err == nil || !strings.Contains(err.Error(), "dirty field is required") {
		t.Fatalf("Parse() error = %v, want required dirty-field rejection", err)
	}
}

func validReportJSON(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "templates", "assessments", "repository-assessment-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
