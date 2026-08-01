package plan

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/status"
)

func TestOutputJSONIncludesContractMetadata(t *testing.T) {
	output := NewOutput(
		config.Spec{Repos: []config.Repo{testRepo()}},
		[]status.Result{{State: status.StateEqual}},
		[]bool{true},
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
		config.Spec{Repos: []config.Repo{testRepo()}},
		[]status.Result{{State: status.StateBehind, Behind: 3}},
		[]bool{true},
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
