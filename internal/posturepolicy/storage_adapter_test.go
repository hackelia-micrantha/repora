package posturepolicy

import (
	"encoding/json"
	"testing"

	"repoctl/internal/posture"
)

func TestStorageConvergencePreservesLocalScope(t *testing.T) {
	ev := posture.Evidence{Source: "git-local", Reference: "fixture"}
	storage := posture.NewStorageInventory("example/project")
	storage.Objects.LooseCount = posture.Observed(int64(0), ev)
	storage.Objects.LooseBytes = posture.Observed(int64(0), ev)
	storage.Objects.PackedCount = posture.Observed(int64(0), ev)
	storage.Objects.PackedBytes = posture.Observed(int64(0), ev)
	storage.Objects.PackCount = posture.Observed(int64(0), ev)
	storage.GitState.Shallow = posture.Observed(true, ev)
	storage.GitState.PromisorConfigured = posture.Unknown[bool](ev)
	data, err := storage.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	inputs, err := ConvergeArtifacts(ArtifactSet{Storage: data})
	if err != nil {
		t.Fatalf("converge: %v", err)
	}
	if inputs.Repository != "example/project" {
		t.Fatalf("wrong repository %q", inputs.Repository)
	}
	if string(inputs.Facts["storage.scope"].Value) != `"local_object_database"` {
		t.Fatalf("missing local-only scope: %+v", inputs.Facts["storage.scope"])
	}
	if inputs.Facts["storage.checkout.promisor_configured"].State != posture.StateUnknown {
		t.Fatal("unknown promisor state became observed")
	}
	if _, exists := inputs.Facts["storage.canonical.bytes"]; exists {
		t.Fatal("invented canonical size fact")
	}
	again, err := inputs.Marshal()
	if err != nil || !json.Valid(again) {
		t.Fatalf("marshal normalized inputs: %v", err)
	}
}

func TestStorageConvergenceRejectsIdentityMismatch(t *testing.T) {
	ev := posture.Evidence{Source: "git-local", Reference: "fixture"}
	storage := posture.NewStorageInventory("example/other")
	storage.Objects.LooseCount = posture.Observed(int64(0), ev)
	storage.Objects.LooseBytes = posture.Observed(int64(0), ev)
	storage.Objects.PackedCount = posture.Observed(int64(0), ev)
	storage.Objects.PackedBytes = posture.Observed(int64(0), ev)
	storage.Objects.PackCount = posture.Observed(int64(0), ev)
	storage.GitState.Shallow = posture.Observed(false, ev)
	storage.GitState.PromisorConfigured = posture.Observed(false, ev)
	inputs := NewInputs("example/project")
	if err := AddStorage(&inputs, storage); err == nil {
		t.Fatal("accepted conflicting operator-asserted repository identities")
	}
}
