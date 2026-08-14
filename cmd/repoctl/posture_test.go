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

func TestPostureInventoryHelpAndUsage(t *testing.T) {
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"posture", "inventory", "--help"})
	})
	if code != 0 || stdout.String() != "usage: repoctl posture inventory OWNER/REPO\n" {
		t.Fatalf("help code=%d output=%q", code, stdout.String())
	}

	var stderr bytes.Buffer
	code = withStderr(t, &stderr, func() int {
		return run([]string{"posture", "inventory"})
	})
	if code != 1 || stderr.String() != "usage: repoctl posture inventory OWNER/REPO\n" {
		t.Fatalf("usage code=%d output=%q", code, stderr.String())
	}
}
