package posture

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCIEnvironmentPostureSchemasAreWellFormed(t *testing.T) {
	for _, schemaPath := range []string{
		"../../schemas/posture-ci-environment-v1.schema.json",
		"../../schemas/posture-ci-environment-profile-v1.schema.json",
	} {
		data, err := os.ReadFile(schemaPath)
		if err != nil {
			t.Fatalf("read CI environment posture schema %s: %v", schemaPath, err)
		}
		var schema any
		if err := json.Unmarshal(data, &schema); err != nil {
			t.Fatalf("parse CI environment posture schema %s: %v", schemaPath, err)
		}
	}
}

func TestCIEnvironmentPostureMarshalPreservesUnknownApplicability(t *testing.T) {
	inventory := CIEnvironmentInventory{
		Kind:                  CIEnvironmentInventoryKind,
		Version:               CIEnvironmentInventoryVersion,
		Repository:            RepositoryIdentity{Provider: "github", FullName: "acme/project"},
		DefaultBranch:         Observed("main"),
		DefaultCommit:         Observed("abc1234"),
		ProfileDeclared:       Observed(false),
		DeclaredApplicability: Unknown[string](),
		FlakePresent:          Observed(true),
		FlakeLockPresent:      Observed(true),
		WorkflowsState:        StateObserved,
		Workflows:             []CIEnvironmentWorkflowFact{},
		ExternalInputs:        []CIExternalInputFact{},
		Evidence:              []Evidence{},
	}
	data, err := inventory.Marshal()
	if err != nil {
		t.Fatalf("marshal CI environment posture: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode CI environment posture: %v", err)
	}
	applicability, ok := decoded["declared_applicability"].(map[string]any)
	if !ok || applicability["state"] != string(StateUnknown) {
		t.Fatalf("unknown applicability fact not preserved: %#v", decoded["declared_applicability"])
	}
}
