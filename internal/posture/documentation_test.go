package posture

import (
	"context"
	"strings"
	"testing"
)

func TestParseDocumentationProfileRejectsDuplicateAndEscapingTargets(t *testing.T) {
	_, err := ParseDocumentationProfile([]byte(`kind: repora.posture-documentation-profile
version: 1
name: bad
readme:
  path: README.md
  sections: [Usage, usage]
  links: []
documents:
  - ../README.md
content_markers: []
`))
	if err == nil {
		t.Fatal("invalid documentation profile was accepted")
	}
}

func TestCollectGitHubDocumentationUsesDeclaredProfileAndRouting(t *testing.T) {
	reader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repos/acme/project"),
		branch:        GitHubBranch{Name: "main", Protected: true, CommitSHA: "abc1234", TreeSHA: "tree123"},
		branchObs:     available("github.branch", "repos/acme/project/branches/main"),
		tree: GitHubTree{Entries: []GitHubTreeEntry{
			{Path: documentationProfilePath, Type: "blob", SHA: "profile"},
			{Path: documentRouterPath, Type: "blob", SHA: "router"},
			{Path: "README.md", Type: "blob", SHA: "readme"},
			{Path: "SECURITY.md", Type: "blob", SHA: "security"},
			{Path: "docs/SUMMARY.md", Type: "blob", SHA: "summary"},
			{Path: "docs/architecture/current-system.md", Type: "blob", SHA: "architecture"},
			{Path: "docs/ci.md", Type: "blob", SHA: "ci"},
		}},
		treeObs: available("github.git_tree", "acme/project:tree123"),
		blobs: map[string][]byte{
			"profile": []byte(`kind: repora.posture-documentation-profile
version: 1
name: service
documents:
  - README.md
  - SECURITY.md
  - docs/SUMMARY.md
readme:
  path: README.md
  sections:
    - Overview
    - Security model
  links:
    - docs/architecture/current-system.md
content_markers:
  - id: current-go-toolchain
    path: docs/ci.md
    contains: Go 1.25.12
`),
			"router": []byte(`version: 1
kind: document-router
trust:
  rules:
    - tier: canonical
      paths:
        - README.md
        - SECURITY.md
        - docs/**
    - tier: generated
      paths:
        - docs/SUMMARY.md
`),
			"readme": []byte("# Project\n\n## Overview\n\nSee [architecture](docs/architecture/current-system.md).\n\n## Security model\n"),
			"ci":     []byte("Current validated toolchain: Go 1.25.12.\n"),
		},
	}

	inventory, err := CollectGitHubDocumentation(context.Background(), reader, "acme/project")
	if err != nil {
		t.Fatalf("collect documentation: %v", err)
	}
	if err := inventory.Validate(); err != nil {
		t.Fatalf("validate documentation inventory: %v", err)
	}
	if !boolValue(t, inventory.ProfileDeclared) || inventory.ProfileName.Value == nil || *inventory.ProfileName.Value != "service" {
		t.Fatalf("profile facts = %#v %#v", inventory.ProfileDeclared, inventory.ProfileName)
	}
	if inventory.DefaultCommit.Value == nil || *inventory.DefaultCommit.Value != "abc1234" {
		t.Fatalf("default commit = %#v", inventory.DefaultCommit)
	}
	if !boolValue(t, inventory.READMEPresent) || len(inventory.READMESections) != 2 || len(inventory.READMELinks) != 1 || len(inventory.ContentMarkers) != 1 {
		t.Fatalf("documentation observations = %#v", inventory)
	}
	for _, section := range inventory.READMESections {
		if !boolValue(t, section.Present) {
			t.Fatalf("section %q was not observed", section.Section)
		}
	}
	if !boolValue(t, inventory.READMELinks[0].Present) || !boolValue(t, inventory.ContentMarkers[0].Present) {
		t.Fatalf("link/marker facts = %#v %#v", inventory.READMELinks[0], inventory.ContentMarkers[0])
	}

	tiers := map[string]string{}
	for _, document := range inventory.Documents {
		if document.TrustTier.State != StateObserved || document.TrustTier.Value == nil {
			t.Fatalf("document trust = %#v", document)
		}
		tiers[document.Path] = *document.TrustTier.Value
	}
	if tiers["README.md"] != "canonical" || tiers["docs/SUMMARY.md"] != "generated" {
		t.Fatalf("document trust tiers = %#v", tiers)
	}
	if !boolValue(t, inventory.RoutingMetadataPresent) || !boolValue(t, inventory.RoutingTrustMetadataUsable) {
		t.Fatalf("router facts = %#v %#v", inventory.RoutingMetadataPresent, inventory.RoutingTrustMetadataUsable)
	}

	data, err := inventory.Marshal()
	if err != nil {
		t.Fatalf("marshal documentation inventory: %v", err)
	}
	if strings.Contains(string(data), "Go 1.25.12") {
		t.Fatal("content marker plaintext leaked into inventory")
	}
}

func TestCollectGitHubDocumentationFallsBackToBaselineOnlyWhenProfileIsKnownAbsent(t *testing.T) {
	reader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repos/acme/project"),
		branch:        GitHubBranch{Name: "main", CommitSHA: "abc1234", TreeSHA: "tree123"},
		branchObs:     available("github.branch", "repos/acme/project/branches/main"),
		tree:          GitHubTree{Entries: []GitHubTreeEntry{}},
		treeObs:       available("github.git_tree", "acme/project:tree123"),
		blobs:         map[string][]byte{},
	}
	inventory, err := CollectGitHubDocumentation(context.Background(), reader, "acme/project")
	if err != nil {
		t.Fatalf("collect documentation: %v", err)
	}
	if boolValue(t, inventory.ProfileDeclared) {
		t.Fatal("absent profile reported as declared")
	}
	if inventory.ProfileName.Value == nil || *inventory.ProfileName.Value != "baseline" {
		t.Fatalf("profile name = %#v", inventory.ProfileName)
	}
	if len(inventory.Documents) != 1 || inventory.Documents[0].Path != "README.md" || boolValue(t, inventory.READMEPresent) {
		t.Fatalf("baseline document facts = %#v", inventory.Documents)
	}
}

func TestCollectGitHubDocumentationDoesNotAssumeBaselineFromTruncatedTree(t *testing.T) {
	reader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repos/acme/project"),
		branch:        GitHubBranch{Name: "main", CommitSHA: "abc1234", TreeSHA: "tree123"},
		branchObs:     available("github.branch", "repos/acme/project/branches/main"),
		tree:          GitHubTree{Truncated: true, Entries: []GitHubTreeEntry{}},
		treeObs:       available("github.git_tree", "acme/project:tree123"),
		blobs:         map[string][]byte{},
	}
	inventory, err := CollectGitHubDocumentation(context.Background(), reader, "acme/project")
	if err != nil {
		t.Fatalf("collect documentation: %v", err)
	}
	if inventory.ProfileDeclared.State != StateUnknown || inventory.ProfileName.State != StateUnknown || inventory.READMEPresent.State != StateUnknown {
		t.Fatalf("truncated profile facts = %#v %#v %#v", inventory.ProfileDeclared, inventory.ProfileName, inventory.READMEPresent)
	}
	if len(inventory.Documents) != 0 {
		t.Fatalf("truncated tree unexpectedly applied baseline: %#v", inventory.Documents)
	}
}

func TestCollectGitHubDocumentationKeepsMalformedDeclaredProfileUnknownWithoutEchoingContent(t *testing.T) {
	const sensitive = "internal-secret-marker"
	reader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repos/acme/project"),
		branch:        GitHubBranch{Name: "main", CommitSHA: "abc1234", TreeSHA: "tree123"},
		branchObs:     available("github.branch", "repos/acme/project/branches/main"),
		tree: GitHubTree{Entries: []GitHubTreeEntry{
			{Path: documentationProfilePath, Type: "blob", SHA: "profile"},
		}},
		treeObs: available("github.git_tree", "acme/project:tree123"),
		blobs: map[string][]byte{
			"profile": []byte("kind: repora.posture-documentation-profile\nversion: 99\nname: " + sensitive + "\nreadme:\n  path: README.md\n"),
		},
	}
	inventory, err := CollectGitHubDocumentation(context.Background(), reader, "acme/project")
	if err != nil {
		t.Fatalf("collect documentation: %v", err)
	}
	if !boolValue(t, inventory.ProfileDeclared) || inventory.ProfileName.State != StateUnknown {
		t.Fatalf("malformed profile facts = %#v %#v", inventory.ProfileDeclared, inventory.ProfileName)
	}
	data, err := inventory.Marshal()
	if err != nil {
		t.Fatalf("marshal malformed profile inventory: %v", err)
	}
	if strings.Contains(string(data), sensitive) {
		t.Fatal("malformed profile content leaked into evidence")
	}
}

func TestDocumentationInventoryRejectsMalformedMarkerDigest(t *testing.T) {
	inventory := newDocumentationInventory("acme/project")
	inventory.DefaultBranch = Observed("main")
	inventory.DefaultCommit = Observed("abc1234")
	inventory.ProfileDeclared = Observed(true)
	inventory.ProfileName = Observed("service")
	inventory.READMEPath = "README.md"
	inventory.READMEPresent = Observed(true)
	inventory.RoutingMetadataPresent = Observed(false)
	inventory.RoutingTrustMetadataUsable = Unknown[bool]()
	inventory.ContentMarkers = append(inventory.ContentMarkers, DocumentationMarkerFact{
		ID:             "marker",
		Path:           "README.md",
		ExpectedSHA256: strings.Repeat("z", 64),
		Present:        Observed(true),
	})
	if err := inventory.Validate(); err == nil {
		t.Fatal("malformed marker digest was accepted")
	}
}

func TestTrustRouterSpecificityPreventsGeneratedDocumentFromBecomingCanonical(t *testing.T) {
	router, err := parseTrustRouter([]byte(`version: 1
kind: document-router
trust:
  rules:
    - tier: canonical
      paths: [docs/**]
    - tier: generated
      paths: [docs/SUMMARY.md]
    - tier: archived
      paths: [docs/archive/**]
`))
	if err != nil {
		t.Fatalf("parse trust router: %v", err)
	}
	for candidate, want := range map[string]string{
		"docs/index.md":        "canonical",
		"docs/SUMMARY.md":      "generated",
		"docs/archive/old.md":  "archived",
		"unclassified/file.md": "unclassified",
	} {
		got, err := router.classify(candidate)
		if err != nil {
			t.Fatalf("classify %s: %v", candidate, err)
		}
		if got != want {
			t.Fatalf("classify %s = %q, want %q", candidate, got, want)
		}
	}
}
