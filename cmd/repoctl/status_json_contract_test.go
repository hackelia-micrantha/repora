package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"repoctl/internal/status"
)

func TestStatusOutputMatchesGoldenContract(t *testing.T) {
	output := status.Output{
		Kind:    status.OutputKind,
		Version: status.OutputVersion,
		Repos: []status.RepositoryResult{{
			ID:        "payments-api",
			UID:       "repo.org.payments-api",
			Canonical: status.RefResult{Ref: "HEAD", Commit: "abc1234"},
			Mirrors: []status.MirrorResult{{
				Target:   "github:org/payments-api",
				Provider: "github",
				Path:     "org/payments-api",
				Ref:      "HEAD",
				Commit:   "def5678",
				State:    status.StateBehind,
				Behind:   3,
			}},
		}},
	}

	got, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "status-v2.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(append(got, '\n'), want) {
		t.Fatalf("status JSON contract changed:\n%s\nwant:\n%s", got, want)
	}
}

func TestEmptyStatusOutputIncludesContractMetadata(t *testing.T) {
	data, err := json.Marshal(status.Output{Kind: status.OutputKind, Version: status.OutputVersion, Repos: []status.RepositoryResult{}})
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
	if got.Kind != status.OutputKind || got.Version != status.OutputVersion || got.Repos == nil {
		t.Fatalf("metadata = %#v, want kind %q version %d and empty repos array", got, status.OutputKind, status.OutputVersion)
	}
}
