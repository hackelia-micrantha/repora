package posture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

	profile := loadRepositoryDocumentationProfile(t)
	if profile.Name != "repora" {
		t.Fatalf("repository documentation profile name = %q", profile.Name)
	}
}

func TestRepositoryDocumentationProfileMatchesCommittedSources(t *testing.T) {
	profile := loadRepositoryDocumentationProfile(t)

	for _, document := range sortedUnique(append(append([]string{}, profile.Documents...), profile.README.Path)) {
		if _, err := os.Stat(filepath.Join("../..", filepath.FromSlash(document))); err != nil {
			t.Fatalf("profile document %s: %v", document, err)
		}
	}

	readmeData, err := os.ReadFile(filepath.Join("../..", filepath.FromSlash(profile.README.Path)))
	if err != nil {
		t.Fatalf("read configured README: %v", err)
	}
	headings := markdownHeadings(readmeData)
	for _, section := range profile.README.Sections {
		if _, ok := headings[normalizeHeading(section)]; !ok {
			t.Fatalf("configured README section %q is not present", section)
		}
	}
	links := markdownRepositoryLinks(profile.README.Path, readmeData)
	for _, target := range profile.README.Links {
		if _, ok := links[target]; !ok {
			t.Fatalf("configured README link target %q is not present", target)
		}
	}

	for _, marker := range profile.ContentMarkers {
		data, err := os.ReadFile(filepath.Join("../..", filepath.FromSlash(marker.Path)))
		if err != nil {
			t.Fatalf("read marker source %s: %v", marker.Path, err)
		}
		if !strings.Contains(string(data), marker.Contains) {
			t.Fatalf("configured content marker %q is stale in %s", marker.ID, marker.Path)
		}
	}

	routerData, err := os.ReadFile("../../.repora/document-router.yaml")
	if err != nil {
		t.Fatalf("read document router: %v", err)
	}
	router, err := parseTrustRouter(routerData)
	if err != nil {
		t.Fatalf("parse routing trust metadata: %v", err)
	}
	for _, document := range sortedUnique(append(append([]string{}, profile.Documents...), profile.README.Path)) {
		tier, err := router.classify(document)
		if err != nil {
			t.Fatalf("classify %s: %v", document, err)
		}
		if tier == "generated" || tier == "archived" {
			t.Fatalf("profile document %s is classified %s and must not be treated as a canonical documentation source", document, tier)
		}
	}
}

func loadRepositoryDocumentationProfile(t *testing.T) DocumentationProfile {
	t.Helper()
	profileData, err := os.ReadFile("../../.repora/posture-documentation.yaml")
	if err != nil {
		t.Fatalf("read repository documentation profile: %v", err)
	}
	profile, err := ParseDocumentationProfile(profileData)
	if err != nil {
		t.Fatalf("parse repository documentation profile: %v", err)
	}
	return profile
}
