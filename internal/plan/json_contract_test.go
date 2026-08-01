package plan

import (
	"encoding/json"
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
