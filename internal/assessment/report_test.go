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
	data = replaceRequired(t, data, `"summary":`, `"unexpected": true, "summary":`)
	if _, err := Parse(data); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Parse() error = %v, want unknown-field rejection", err)
	}
}

func TestParseRejectsUnknownEvidenceLink(t *testing.T) {
	data := validReportJSON(t)
	data = replaceRequired(t, data, `"evidence_ids": []`, `"evidence_ids": ["missing"]`)
	if _, err := Parse(data); err == nil || !strings.Contains(err.Error(), "unknown evidence id") {
		t.Fatalf("Parse() error = %v, want unknown evidence rejection", err)
	}
}

func TestParseRejectsMissingDirtyField(t *testing.T) {
	data := validReportJSON(t)
	data = replaceRequired(t, data, `, "dirty": false`, "")
	if _, err := Parse(data); err == nil || !strings.Contains(err.Error(), "dirty field is required") {
		t.Fatalf("Parse() error = %v, want required dirty-field rejection", err)
	}
}

func TestParseRejectsNullValues(t *testing.T) {
	data := validReportJSON(t)
	data = replaceRequired(t, data, `"created_at": "1970-01-01T00:00:00Z"`, `"created_at": null`)
	if _, err := Parse(data); err == nil || !strings.Contains(err.Error(), "must not be null") {
		t.Fatalf("Parse() error = %v, want null rejection", err)
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

func replaceRequired(t *testing.T, data []byte, old, replacement string) []byte {
	t.Helper()
	if !strings.Contains(string(data), old) {
		t.Fatalf("test fixture does not contain %q", old)
	}
	return []byte(strings.Replace(string(data), old, replacement, 1))
}
