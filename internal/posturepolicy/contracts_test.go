package posturepolicy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPosturePolicySchemasAreWellFormedJSON(t *testing.T) {
	for _, name := range []string{
		"posture-policy-profile-v1.schema.json",
		"posture-policy-inputs-v1.schema.json",
		"posture-report-v1.schema.json",
	} {
		path := filepath.Join("..", "..", "schemas", name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		var schema map[string]any
		if err := json.Unmarshal(data, &schema); err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		if schema["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
			t.Fatalf("%s uses unexpected JSON Schema dialect", name)
		}
	}
}
