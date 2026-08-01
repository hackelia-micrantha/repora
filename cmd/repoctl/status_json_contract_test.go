package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/status"
)

func TestStatusOutputMatchesGoldenContract(t *testing.T) {
	spec := config.Spec{Repos: []config.Repo{{
		ID:        "payments-api",
		UID:       "repo.org.payments-api",
		Canonical: config.Endpoint{Provider: "gitlab"},
		Mirrors:   []config.Endpoint{{Provider: "github"}},
	}}}
	output := newJSONOutput(spec, []status.Result{{
		State:     status.StateBehind,
		Canonical: "abc1234",
		Mirror:    "def5678",
		Behind:    3,
	}}, []bool{true})

	got, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "status-v1.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(append(got, '\n'), want) {
		t.Fatalf("status JSON contract changed:\n%s\nwant:\n%s", got, want)
	}
}

func TestEmptyStatusOutputIncludesContractMetadata(t *testing.T) {
	data, err := json.Marshal(newJSONOutput(config.Spec{}, nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Kind    string `json:"kind"`
		Version int    `json:"version"`
		Repos   []any  `json:"repos"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Kind != statusOutputKind || got.Version != statusOutputVersion || got.Repos == nil {
		t.Fatalf("metadata = %#v, want kind %q version %d and empty repos array", got, statusOutputKind, statusOutputVersion)
	}
}
