package apply

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"repoctl/internal/status"
)

func TestDetailedOutputJSONMatchesGoldenContract(t *testing.T) {
	output := NewDetailedOutput([]DetailedResult{{
		ID: "payments-api", UID: "repo.org.payments-api", State: status.StateDiverged, Applied: false, DryRun: false,
		Actions: []DetailedAction{
			{
				Type: "PUSH_BRANCH", Source: "gitlab:org/payments-api/main", Target: "github:org/payments-api/main",
				Before: "2222222222222222222222222222222222222222", Desired: "1111111111111111111111111111111111111111",
				After: "1111111111111111111111111111111111111111", Outcome: "APPLIED",
			},
			{
				Type: "PUSH_BRANCH", Source: "gitlab:org/payments-api/main", Target: "gitlab:backup/payments-api/main", Force: true,
				Before: "3333333333333333333333333333333333333333", Desired: "1111111111111111111111111111111111111111",
				Outcome: "FAILED", Error: "mirror unavailable",
			},
		},
		Journal: &JournalReferences{
			ExecutionID: "run-001",
			Intent:      ".repora/journal/repo.org.payments-api--run-001--intent.json",
			Result:      ".repora/journal/repo.org.payments-api--run-001--result.json",
		},
		Error: "one mirror action failed",
	}})
	got, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/apply-v3.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want)) {
		t.Fatalf("apply v3 contract changed:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
