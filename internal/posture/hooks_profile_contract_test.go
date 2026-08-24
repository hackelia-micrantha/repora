package posture

import (
	"encoding/json"
	"os"
	"testing"
)

func TestHooksPostureProfileSchemaIsWellFormed(t *testing.T) {
	data, err := os.ReadFile("../../schemas/posture-hooks-profile-v1.schema.json")
	if err != nil {
		t.Fatalf("read hooks posture profile schema: %v", err)
	}
	var schema any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("parse hooks posture profile schema: %v", err)
	}
}
