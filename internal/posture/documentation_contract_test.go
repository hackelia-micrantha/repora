package posture

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDocumentationSchemasAndRepositoryProfileAreWellFormed(t *testing.T) {
	for _, schemaPath := range []string{
		"../../schemas/posture-documentation-v1.schema.json",
		"../../schemas/posture-documentation-profile-v1.schema.json",
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

	profileData, err := os.ReadFile("../../.repora/posture-documentation.yaml")
	if err != nil {
		t.Fatalf("read repository documentation profile: %v", err)
	}
	profile, err := ParseDocumentationProfile(profileData)
	if err != nil {
		t.Fatalf("parse repository documentation profile: %v", err)
	}
	if profile.Name != "repora" {
		t.Fatalf("repository documentation profile name = %q", profile.Name)
	}
}
