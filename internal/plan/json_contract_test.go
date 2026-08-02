package plan

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"repoctl/internal/status"
)

func TestOutputJSONIncludesContractMetadata(t *testing.T) {
	output := NewOutput(
		[]ReconciliationPlan{{ID: "payments-api", UID: "repo.org.payments-api", Actions: []PlannedAction{}}},
		[]status.Result{{State: status.StateEqual}},
	)
	data, err := json.Marshal(output)
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
	output := NewOutput(
		[]ReconciliationPlan{{
			ID:  "payments-api",
			UID: "repo.org.payments-api",
			Actions: []PlannedAction{{
				Type:   ActionPushBranch,
				Target: Remote{Provider: "github", Name: "mirror", Branch: "main"},
			}},
		}},
		[]status.Result{{State: status.StateBehind, Behind: 3}},
	)
	got, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/plan-v1.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want)) {
		t.Fatalf("plan JSON contract changed:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
