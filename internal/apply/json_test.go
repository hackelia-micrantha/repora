package apply

import (
	"encoding/json"
	"testing"
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
