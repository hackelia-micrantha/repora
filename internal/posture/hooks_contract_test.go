package posture

import (
	"encoding/json"
	"os"
	"testing"
)

func TestHooksPostureSchemaIsWellFormed(t *testing.T) {
	data, err := os.ReadFile("../../schemas/posture-hooks-v1.schema.json")
	if err != nil {
		t.Fatalf("read hooks posture schema: %v", err)
	}
	var schema any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("parse hooks posture schema: %v", err)
	}
}

func TestHooksPostureMarshalPreservesUnknownFacts(t *testing.T) {
	inventory := HooksInventory{
		Kind:             HooksInventoryKind,
		Version:          HooksInventoryVersion,
		Repository:       RepositoryIdentity{Provider: "github", FullName: "acme/project"},
		DefaultBranch:    Observed("main"),
		DefaultCommit:    Observed("abc1234"),
		ProfileDeclared:  Observed(false),
		Manager:          Observed("none"),
		Entrypoints:      []HookEntrypointFact{},
		RequiredChecks:   []LocalCheckFact{},
		BootstrapPresent: Unknown[bool](),
		BypassPresent:    Unknown[bool](),
		Evidence:         []Evidence{},
	}
	data, err := inventory.Marshal()
	if err != nil {
		t.Fatalf("marshal hooks posture: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode hooks posture: %v", err)
	}
	if decoded["kind"] != HooksInventoryKind || decoded["version"] != float64(HooksInventoryVersion) {
		t.Fatalf("unexpected envelope: %#v", decoded)
	}
	bootstrap, ok := decoded["bootstrap_instructions_present"].(map[string]any)
	if !ok || bootstrap["state"] != string(StateUnknown) {
		t.Fatalf("unknown bootstrap fact not preserved: %#v", decoded["bootstrap_instructions_present"])
	}
}
