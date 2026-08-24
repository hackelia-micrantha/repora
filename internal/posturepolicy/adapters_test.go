package posturepolicy

import (
	"encoding/json"
	"testing"

	"repoctl/internal/posture"
)

func TestAddInventoryPreservesFactStates(t *testing.T) {
	inventory := posture.NewInventory("acme/project")
	inventory.RepositoryFacts = posture.RepositoryFacts{
		DefaultBranch:              posture.Observed("main", posture.Evidence{Source: "github", Reference: "repo"}),
		DefaultBranchProtected:     posture.Unknown[bool](posture.Evidence{Source: "github", Reference: "protection"}),
		RequiredStatusChecks:       posture.Unavailable[[]string](posture.Evidence{Source: "github", Reference: "protection"}),
		RequiredReviews:            posture.Observed(1),
		ForcePushProtected:         posture.Observed(true),
		DeletionProtected:          posture.Observed(true),
		CODEOWNERSPresent:          posture.Observed(true),
		SecurityMDPresent:          posture.Observed(true),
		LicensePresent:             posture.Observed(true),
		IssueTemplatePresent:       posture.Observed(true),
		PullRequestTemplatePresent: posture.Observed(true),
		DependencyAutomation:       posture.Observed([]string{"dependabot"}),
		WorkflowPaths:              posture.Observed([]string{}),
	}
	inventory.WorkflowsState = posture.StateObserved

	inputs := NewInputs("acme/project")
	if err := AddInventory(&inputs, inventory); err != nil {
		t.Fatal(err)
	}
	if got := inputs.Facts["repository.default_branch_protected"].State; got != posture.StateUnknown {
		t.Fatalf("protected state = %q", got)
	}
	if got := inputs.Facts["repository.required_status_checks"].State; got != posture.StateUnavailable {
		t.Fatalf("checks state = %q", got)
	}
	if got := string(inputs.Facts["repository.default_branch"].Value); got != `"main"` {
		t.Fatalf("default branch value = %s", got)
	}
}

func TestAdaptersRejectRepositoryMixing(t *testing.T) {
	inventory := posture.NewInventory("other/project")
	inventory.RepositoryFacts = posture.RepositoryFacts{
		DefaultBranch:              posture.Observed("main"),
		DefaultBranchProtected:     posture.Observed(true),
		RequiredStatusChecks:       posture.Observed([]string{}),
		RequiredReviews:            posture.Observed(1),
		ForcePushProtected:         posture.Observed(true),
		DeletionProtected:          posture.Observed(true),
		CODEOWNERSPresent:          posture.Observed(true),
		SecurityMDPresent:          posture.Observed(true),
		LicensePresent:             posture.Observed(true),
		IssueTemplatePresent:       posture.Observed(true),
		PullRequestTemplatePresent: posture.Observed(true),
		DependencyAutomation:       posture.Observed([]string{}),
		WorkflowPaths:              posture.Observed([]string{}),
	}
	inventory.WorkflowsState = posture.StateObserved
	inputs := NewInputs("acme/project")
	if err := AddInventory(&inputs, inventory); err == nil {
		t.Fatal("expected repository mismatch")
	}
}

func TestDocumentationHooksCommitsAndMirrorsAdaptWithoutInference(t *testing.T) {
	inputs := NewInputs("acme/project")
	docs := posture.DocumentationInventory{
		Kind:                       posture.DocumentationInventoryKind,
		Version:                    posture.DocumentationInventoryVersion,
		Repository:                 posture.RepositoryIdentity{Provider: "github", FullName: "acme/project"},
		DefaultBranch:              posture.Observed("main"),
		DefaultCommit:              posture.Observed("abc1234"),
		ProfileDeclared:            posture.Observed(false),
		ProfileName:                posture.Observed("baseline"),
		READMEPath:                 "README.md",
		READMEPresent:              posture.Observed(true),
		Documents:                  []posture.DocumentationDocumentFact{},
		READMESections:             []posture.DocumentationSectionFact{},
		READMELinks:                []posture.DocumentationLinkFact{},
		ContentMarkers:             []posture.DocumentationMarkerFact{},
		RoutingMetadataPresent:     posture.Observed(false),
		RoutingTrustMetadataUsable: posture.Unknown[bool](posture.Evidence{Source: "routing", Reference: "profile"}),
		Evidence:                   []posture.Evidence{},
	}
	if err := AddDocumentation(&inputs, docs); err != nil {
		t.Fatal(err)
	}

	hooks := posture.HooksInventory{
		Kind:             posture.HooksInventoryKind,
		Version:          posture.HooksInventoryVersion,
		Repository:       posture.RepositoryIdentity{Provider: "github", FullName: "acme/project"},
		DefaultBranch:    posture.Observed("main"),
		DefaultCommit:    posture.Observed("abc1234"),
		ProfileDeclared:  posture.Observed(false),
		Manager:          posture.Observed("none"),
		Entrypoints:      []posture.HookEntrypointFact{},
		RequiredChecks:   []posture.LocalCheckFact{},
		BootstrapPresent: posture.Unknown[bool](),
		BypassPresent:    posture.Unknown[bool](),
		Evidence:         []posture.Evidence{},
	}
	if err := AddHooks(&inputs, hooks); err != nil {
		t.Fatal(err)
	}

	commits := posture.CommitInventory{
		Kind:                       posture.CommitInventoryKind,
		Version:                    posture.CommitInventoryVersion,
		Repository:                 posture.RepositoryIdentity{Provider: "github", FullName: "acme/project"},
		DefaultBranch:              posture.Observed("main"),
		DefaultCommit:              posture.Observed("abc1234"),
		ProfileDeclared:            posture.Observed(false),
		HistoryLimit:               posture.Observed(20),
		HistoryTruncated:           posture.Observed(false),
		FileCountThreshold:         posture.Observed(50),
		ChangedLinesThreshold:      posture.Observed(1000),
		SensitivePathPatterns:      posture.Observed([]string{}),
		Commits:                    []posture.CommitHistoryFact{},
		SignedTagCount:             posture.Unknown[int](posture.Evidence{Source: "scope", Reference: "tags"}),
		UnsignedTagCount:           posture.Unknown[int](posture.Evidence{Source: "scope", Reference: "tags"}),
		ReleaseBoundaryChangeCount: posture.Unknown[int](posture.Evidence{Source: "scope", Reference: "releases"}),
		Evidence:                   []posture.Evidence{},
	}
	if err := AddCommits(&inputs, commits); err != nil {
		t.Fatal(err)
	}

	mirrorRepo := posture.MirrorRepositoryFacts{
		ID:        "project",
		UID:       "repo.project",
		Mode:      posture.Observed("mirror"),
		Direction: posture.Observed("canonical_to_mirror"),
		Canonical: posture.MirrorCanonicalFacts{
			Identity:                   posture.MirrorEndpointIdentity{Provider: "gitlab", Path: "acme/project"},
			DefaultBranch:              posture.Observed("main"),
			Commit:                     posture.Observed("abc1234"),
			Visibility:                 posture.Unavailable[string](posture.Evidence{Source: "gitlab", Reference: "metadata"}),
			CurrentActorPushPermission: posture.Unavailable[bool](posture.Evidence{Source: "gitlab", Reference: "metadata"}),
		},
		Mirrors:  []posture.MirrorTargetFacts{},
		Evidence: []posture.Evidence{},
	}
	if err := AddMirrorRepository(&inputs, mirrorRepo); err != nil {
		t.Fatal(err)
	}

	if inputs.Facts["documentation.routing_trust_metadata_usable"].State != posture.StateUnknown {
		t.Fatal("documentation unknown state was not preserved")
	}
	if inputs.Facts["commits.signed_tag_count"].State != posture.StateUnknown {
		t.Fatal("commit out-of-scope state was not preserved")
	}
	if inputs.Facts["mirrors.canonical.visibility"].State != posture.StateUnavailable {
		t.Fatal("mirror unavailable state was not preserved")
	}
	if got := inputs.Facts["mirrors.target_count"].Value; string(got) != "0" {
		t.Fatalf("mirror target count = %s", got)
	}
	if _, err := inputs.Marshal(); err != nil {
		t.Fatal(err)
	}
}

func TestDuplicateAdaptedFactIsRejected(t *testing.T) {
	inputs := NewInputs("acme/project")
	inputs.Facts["commits.observed_count"] = FactInput{State: posture.StateObserved, Value: json.RawMessage(`0`), Evidence: []posture.Evidence{}}
	commits := posture.CommitInventory{
		Kind:                       posture.CommitInventoryKind,
		Version:                    posture.CommitInventoryVersion,
		Repository:                 posture.RepositoryIdentity{Provider: "github", FullName: "acme/project"},
		DefaultBranch:              posture.Observed("main"),
		DefaultCommit:              posture.Observed("abc1234"),
		ProfileDeclared:            posture.Observed(false),
		HistoryLimit:               posture.Observed(20),
		HistoryTruncated:           posture.Observed(false),
		FileCountThreshold:         posture.Observed(50),
		ChangedLinesThreshold:      posture.Observed(1000),
		SensitivePathPatterns:      posture.Observed([]string{}),
		Commits:                    []posture.CommitHistoryFact{},
		SignedTagCount:             posture.Unknown[int](),
		UnsignedTagCount:           posture.Unknown[int](),
		ReleaseBoundaryChangeCount: posture.Unknown[int](),
		Evidence:                   []posture.Evidence{},
	}
	if err := AddCommits(&inputs, commits); err == nil {
		t.Fatal("expected duplicate fact to fail")
	}
}
