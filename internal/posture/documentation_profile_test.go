package posture

import (
	"context"
	"testing"
)

func TestParseDocumentationProfileAllowsOptionalObservationCategories(t *testing.T) {
	profile, err := ParseDocumentationProfile([]byte(`kind: repora.posture-documentation-profile
version: 1
name: minimal
readme:
  path: README.md
`))
	if err != nil {
		t.Fatalf("parse minimal profile: %v", err)
	}
	if profile.Name != "minimal" || profile.README.Path != "README.md" {
		t.Fatalf("minimal profile = %#v", profile)
	}
	if len(profile.Documents) != 0 || len(profile.README.Sections) != 0 || len(profile.README.Links) != 0 || len(profile.ContentMarkers) != 0 {
		t.Fatalf("optional categories were not empty: %#v", profile)
	}
}

func TestCollectGitHubDocumentationDistinguishesMissingSectionFromUnavailableContent(t *testing.T) {
	base := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repos/acme/project"),
		branch:        GitHubBranch{Name: "main", CommitSHA: "abc1234", TreeSHA: "tree123"},
		branchObs:     available("github.branch", "repos/acme/project/branches/main"),
		tree: GitHubTree{Entries: []GitHubTreeEntry{
			{Path: documentationProfilePath, Type: "blob", SHA: "profile"},
			{Path: "README.md", Type: "blob", SHA: "readme"},
		}},
		treeObs: available("github.git_tree", "acme/project:tree123"),
		blobs: map[string][]byte{
			"profile": []byte(`kind: repora.posture-documentation-profile
version: 1
name: sections
readme:
  path: README.md
  sections:
    - Overview
    - Security
`),
			"readme": []byte("# Project\n\n## Overview\n"),
		},
	}

	observed, err := CollectGitHubDocumentation(context.Background(), base, "acme/project")
	if err != nil {
		t.Fatalf("collect observed documentation: %v", err)
	}
	if len(observed.READMESections) != 2 {
		t.Fatalf("sections = %#v", observed.READMESections)
	}
	sections := map[string]Fact[bool]{}
	for _, fact := range observed.READMESections {
		sections[fact.Section] = fact.Present
	}
	if !boolValue(t, sections["Overview"]) {
		t.Fatal("Overview should be observed present")
	}
	if sections["Security"].State != StateObserved || sections["Security"].Value == nil || *sections["Security"].Value {
		t.Fatalf("missing Security section = %#v", sections["Security"])
	}

	base.blobObs = map[string]ReadObservation{
		"readme": unavailable("github.blob", "acme/project:readme"),
	}
	unavailableInventory, err := CollectGitHubDocumentation(context.Background(), base, "acme/project")
	if err != nil {
		t.Fatalf("collect unavailable documentation: %v", err)
	}
	for _, section := range unavailableInventory.READMESections {
		if section.Present.State != StateUnavailable || section.Present.Value != nil {
			t.Fatalf("unavailable section %q = %#v", section.Section, section.Present)
		}
	}
}
