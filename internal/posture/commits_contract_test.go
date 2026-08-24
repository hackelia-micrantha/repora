package posture

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCommitPostureSchemasAreWellFormed(t *testing.T) {
	for _, schemaPath := range []string{
		"../../schemas/posture-commits-v1.schema.json",
		"../../schemas/posture-commits-profile-v1.schema.json",
	} {
		data, err := os.ReadFile(schemaPath)
		if err != nil {
			t.Fatalf("read %s: %v", schemaPath, err)
		}
		var schema any
		if err := json.Unmarshal(data, &schema); err != nil {
			t.Fatalf("parse %s: %v", schemaPath, err)
		}
	}
}

func TestCommitPostureMarshalPreservesUnknownOutOfScopeFacts(t *testing.T) {
	inventory := newCommitInventory("acme/project")
	inventory.DefaultBranch = Observed("main")
	inventory.DefaultCommit = Observed("abc1234")
	inventory.ProfileDeclared = Observed(false)
	inventory.HistoryLimit = Observed(20)
	inventory.HistoryTruncated = Observed(false)
	inventory.FileCountThreshold = Observed(50)
	inventory.ChangedLinesThreshold = Observed(1000)
	inventory.SensitivePathPatterns = Observed([]string{})

	data, err := inventory.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded["kind"] != CommitInventoryKind || decoded["version"] != float64(CommitInventoryVersion) {
		t.Fatalf("unexpected envelope: %#v", decoded)
	}
	for _, field := range []string{"signed_tag_count", "unsigned_tag_count", "release_boundary_change_count"} {
		fact, ok := decoded[field].(map[string]any)
		if !ok || fact["state"] != string(StateUnknown) {
			t.Fatalf("%s = %#v", field, decoded[field])
		}
	}
}
