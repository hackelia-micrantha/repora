package apply

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"repoctl/internal/status"
)

func TestOutputJSONIncludesContractMetadata(t *testing.T) {
	data, err := json.Marshal(Output{Results: []Result{}})
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Kind    string `json:"kind"`
		Version int    `json:"version"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Kind != OutputKind || got.Version != OutputVersion {
		t.Fatalf("metadata = %#v, want kind %q version %d", got, OutputKind, OutputVersion)
	}
}

func TestOutputJSONMatchesGoldenContract(t *testing.T) {
	output := Output{Results: []Result{{
		ID:      "payments-api",
		UID:     "repo.org.payments-api",
		State:   status.StateBehind,
		DryRun:  true,
		Actions: []Action{{Type: "PUSH_BRANCH", Source: "canonical/main", Target: "github/main"}},
		Journal: &JournalReferences{
			ExecutionID: "run-001",
			Intent:      ".repora/journal/repo.org.payments-api--run-001--intent.json",
			Result:      ".repora/journal/repo.org.payments-api--run-001--result.json",
		},
	}}}
	got, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/apply-v2.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want)) {
		t.Fatalf("apply JSON contract changed:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
