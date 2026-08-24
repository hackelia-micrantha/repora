package main

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"repoctl/internal/posture"
)

func validPostureInventory(fullName string) posture.Inventory {
	inventory := posture.NewInventory(fullName)
	inventory.RepositoryFacts = posture.RepositoryFacts{
		DefaultBranch:              posture.Observed("main"),
		DefaultBranchProtected:     posture.Observed(true),
		RequiredStatusChecks:       posture.Observed([]string{"ci"}),
		RequiredReviews:            posture.Observed(1),
		ForcePushProtected:         posture.Observed(true),
		DeletionProtected:          posture.Observed(true),
		CODEOWNERSPresent:          posture.Observed(true),
		SecurityMDPresent:          posture.Observed(false),
		LicensePresent:             posture.Observed(true),
		IssueTemplatePresent:       posture.Observed(false),
		PullRequestTemplatePresent: posture.Observed(true),
		DependencyAutomation:       posture.Observed([]string{"dependabot"}),
		WorkflowPaths:              posture.Observed([]string{".github/workflows/ci.yml"}),
	}
	inventory.WorkflowsState = posture.StateObserved
	return inventory
}

func validDocumentationInventory(fullName string) posture.DocumentationInventory {
	return posture.DocumentationInventory{
		Kind:            posture.DocumentationInventoryKind,
		Version:         posture.DocumentationInventoryVersion,
		Repository:      posture.RepositoryIdentity{Provider: "github", FullName: fullName},
		DefaultBranch:   posture.Observed("main"),
		DefaultCommit:   posture.Observed("abc1234"),
		ProfileDeclared: posture.Observed(false),
		ProfileName:     posture.Observed("baseline"),
		READMEPath:      "README.md",
		READMEPresent:   posture.Observed(true),
		Documents: []posture.DocumentationDocumentFact{
			{Path: "README.md", Present: posture.Observed(true), TrustTier: posture.Unknown[string]()},
		},
		READMESections:             []posture.DocumentationSectionFact{},
		READMELinks:                []posture.DocumentationLinkFact{},
		ContentMarkers:             []posture.DocumentationMarkerFact{},
		RoutingMetadataPresent:     posture.Observed(false),
		RoutingTrustMetadataUsable: posture.Unknown[bool](),
		Evidence:                   []posture.Evidence{},
	}
}

func validHooksInventory(fullName string) posture.HooksInventory {
	return posture.HooksInventory{
		Kind:             posture.HooksInventoryKind,
		Version:          posture.HooksInventoryVersion,
		Repository:       posture.RepositoryIdentity{Provider: "github", FullName: fullName},
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
}

func TestPostureInventoryCommandEmitsVersionedJSONAndUsesEnvironmentToken(t *testing.T) {
	old := collectGitHubPosture
	t.Cleanup(func() { collectGitHubPosture = old })
	t.Setenv("GITHUB_TOKEN", "test-token")
	var gotRepo, gotToken string
	collectGitHubPosture = func(_ context.Context, fullName, token string) (posture.Inventory, error) {
		gotRepo, gotToken = fullName, token
		return validPostureInventory(fullName), nil
	}
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"posture", "inventory", "acme/project"})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if gotRepo != "acme/project" || gotToken != "test-token" {
		t.Fatalf("collector inputs repo=%q token=%q", gotRepo, gotToken)
	}
	var decoded posture.Inventory
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	if decoded.Kind != posture.InventoryKind || decoded.Version != posture.InventoryVersion || decoded.Repository.FullName != "acme/project" {
		t.Fatalf("inventory envelope = %#v", decoded)
	}
}

func TestPostureDocumentationCommandEmitsVersionedJSONAndUsesEnvironmentToken(t *testing.T) {
	old := collectGitHubDocumentationPosture
	t.Cleanup(func() { collectGitHubDocumentationPosture = old })
	t.Setenv("GITHUB_TOKEN", "docs-token")
	var gotRepo, gotToken string
	collectGitHubDocumentationPosture = func(_ context.Context, fullName, token string) (posture.DocumentationInventory, error) {
		gotRepo, gotToken = fullName, token
		return validDocumentationInventory(fullName), nil
	}
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"posture", "docs", "acme/project"})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if gotRepo != "acme/project" || gotToken != "docs-token" {
		t.Fatalf("collector inputs repo=%q token=%q", gotRepo, gotToken)
	}
	var decoded posture.DocumentationInventory
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	if decoded.Kind != posture.DocumentationInventoryKind || decoded.Version != posture.DocumentationInventoryVersion || decoded.Repository.FullName != "acme/project" {
		t.Fatalf("documentation inventory envelope = %#v", decoded)
	}
}

func TestPostureHooksCommandEmitsVersionedJSONAndUsesEnvironmentToken(t *testing.T) {
	old := collectGitHubHooksPosture
	t.Cleanup(func() { collectGitHubHooksPosture = old })
	t.Setenv("GITHUB_TOKEN", "hooks-token")
	var gotRepo, gotToken string
	collectGitHubHooksPosture = func(_ context.Context, fullName, token string) (posture.HooksInventory, error) {
		gotRepo, gotToken = fullName, token
		return validHooksInventory(fullName), nil
	}
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"posture", "hooks", "acme/project"})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if gotRepo != "acme/project" || gotToken != "hooks-token" {
		t.Fatalf("collector inputs repo=%q token=%q", gotRepo, gotToken)
	}
	var decoded posture.HooksInventory
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	if decoded.Kind != posture.HooksInventoryKind || decoded.Version != posture.HooksInventoryVersion || decoded.Repository.FullName != "acme/project" {
		t.Fatalf("hooks inventory envelope = %#v", decoded)
	}
}

func TestPostureInventoryFallsBackToGHToken(t *testing.T) {
	old := collectGitHubPosture
	t.Cleanup(func() { collectGitHubPosture = old })
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "fallback-token")
	var gotToken string
	collectGitHubPosture = func(_ context.Context, fullName, token string) (posture.Inventory, error) {
		gotToken = token
		return validPostureInventory(fullName), nil
	}
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"posture", "inventory", "acme/project"})
	})
	if code != 0 || gotToken != "fallback-token" {
		t.Fatalf("code=%d token=%q", code, gotToken)
	}
}

func TestPostureHelpAndUsage(t *testing.T) {
	for _, subcommand := range []string{"inventory", "docs", "hooks"} {
		var stdout bytes.Buffer
		code := withStdout(t, &stdout, func() int {
			return run([]string{"posture", subcommand, "--help"})
		})
		want := "usage: repoctl posture " + subcommand + " OWNER/REPO\n"
		if code != 0 || stdout.String() != want {
			t.Fatalf("%s help code=%d output=%q want=%q", subcommand, code, stdout.String(), want)
		}
	}
	var stderr bytes.Buffer
	code := withStderr(t, &stderr, func() int {
		return run([]string{"posture", "inventory"})
	})
	want := "usage: repoctl posture inventory OWNER/REPO\n       repoctl posture docs OWNER/REPO\n       repoctl posture hooks OWNER/REPO\n       repoctl posture mirrors -f repora.yaml\n"
	if code != 1 || stderr.String() != want {
		t.Fatalf("usage code=%d output=%q want=%q", code, stderr.String(), want)
	}
}
