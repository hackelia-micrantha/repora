package posturepolicy

import (
	"encoding/json"
	"os"
	"testing"
)

func TestPolicyV2SchemasAreWellFormed(t *testing.T) {
	for _, path := range []string{
		"../../schemas/posture-policy-profile-v2.schema.json",
		"../../schemas/posture-report-v2.schema.json",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var document any
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
	}
}
