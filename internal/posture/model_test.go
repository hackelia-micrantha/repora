package posture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBooleanFactDistinguishesFalseUnknownAndUnavailable(t *testing.T) {
	observedFalse := Observed(false, Evidence{Source: "test", Reference: "false"})
	unknown := Unknown[bool](Evidence{Source: "test", Reference: "unknown"})
	unavailable := Unavailable[bool](Evidence{Source: "test", Reference: "unavailable"})
	if observedFalse.State != StateObserved || observedFalse.Value == nil || *observedFalse.Value {
		t.Fatalf("observed false = %#v", observedFalse)
	}
	if unknown.State != StateUnknown || unknown.Value != nil {
		t.Fatalf("unknown = %#v", unknown)
	}
	if unavailable.State != StateUnavailable || unavailable.Value != nil {
		t.Fatalf("unavailable = %#v", unavailable)
	}
}

func TestPostureInventorySchemaIsValidJSON(t *testing.T) {
	path := filepath.Join("..", "..", "schemas", "posture-inventory-v1.schema.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read posture schema: %v", err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("parse posture schema: %v", err)
	}
	if schema["$id"] != "https://repora.io/schemas/posture-inventory-v1.schema.json" {
		t.Fatalf("schema id = %#v", schema["$id"])
	}
}
