package journal

import (
	"strings"
	"testing"
)

func TestParseManagedArtifactRejectsNullUnknownAndTrailingJSON(t *testing.T) {
	plan := managedJournalPlan(t, 1)
	record, err := ManagedArtifactIntent("run-json", plan)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := record.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	// Intent omits failure_stage, so inject an explicit null before repositories.
	nullValue := strings.Replace(string(encoded), "\"repositories\":", "\"failure_stage\": null,\n  \"repositories\":", 1)
	if nullValue == string(encoded) {
		t.Fatal("failed to inject null fixture")
	}
	if _, err := ParseManagedArtifact([]byte(nullValue)); err == nil || !strings.Contains(err.Error(), "must not be null") {
		t.Fatalf("null parse error = %v", err)
	}

	unknown := strings.Replace(string(encoded), "\"repositories\":", "\"unexpected\": true,\n  \"repositories\":", 1)
	if unknown == string(encoded) {
		t.Fatal("failed to inject unknown-field fixture")
	}
	if _, err := ParseManagedArtifact([]byte(unknown)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown-field parse error = %v", err)
	}

	trailing := append(append([]byte(nil), encoded...), []byte(` {}`)...)
	if _, err := ParseManagedArtifact(trailing); err == nil || !strings.Contains(err.Error(), "trailing JSON value") {
		t.Fatalf("trailing parse error = %v", err)
	}
}
