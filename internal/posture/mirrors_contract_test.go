package posture

import (
	"encoding/json"
	"os"
	"testing"
)

func TestMirrorPostureSchemaIsWellFormed(t *testing.T) {
	data, err := os.ReadFile("../../schemas/posture-mirrors-v1.schema.json")
	if err != nil {
		t.Fatalf("read mirror posture schema: %v", err)
	}
	var schema any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("parse mirror posture schema: %v", err)
	}
}

func TestMirrorPostureMarshalPreservesUnknownScopeFacts(t *testing.T) {
	inventory := NewMirrorInventory()
	inventory.Repos = append(inventory.Repos, MirrorRepositoryFacts{
		ID: "example", UID: "repo.example",
		Mode:      Observed("mirror"),
		Direction: Observed("canonical_to_mirror"),
		Canonical: MirrorCanonicalFacts{
			Identity: MirrorEndpointIdentity{Provider: "gitlab", Path: "acme/example"},
			DefaultBranch: Observed("main"), Commit: Observed("abc1234"),
			Visibility: Unavailable[string](), CurrentActorPushPermission: Unavailable[bool](),
		},
		Mirrors: []MirrorTargetFacts{
			{
				Identity: MirrorEndpointIdentity{Provider: "github", Path: "acme/example"},
				CacheRemote: Observed("mirror-0"), DefaultBranch: Observed("main"), DefaultBranchDrift: Observed(false),
				Commit: Observed("abc1234"), Divergence: Observed("EQUAL"), Ahead: Observed(0), Behind: Observed(0),
				Visibility: Unknown[string](), CurrentActorPushPermission: Unknown[bool](), TagDrift: Unknown[bool](), ReleaseDrift: Unknown[bool](),
			},
		},
		Evidence: []Evidence{},
	})

	data, err := inventory.Marshal()
	if err != nil {
		t.Fatalf("marshal mirror posture: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode mirror posture: %v", err)
	}
	if decoded["kind"] != MirrorInventoryKind || decoded["version"] != float64(MirrorInventoryVersion) {
		t.Fatalf("unexpected envelope: %#v", decoded)
	}
}
