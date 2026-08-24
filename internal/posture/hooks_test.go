package posture

import (
	"context"
	"testing"
)

func TestCollectGitHubHooksDetectsCustomHooksAndCICoverage(t *testing.T) {
	reader := fakeGitHubReader{
		repository: GitHubRepository{DefaultBranch: "main"}, repositoryObs: available("github.repository", "repos/acme/project"),
		branch: GitHubBranch{Name: "main", CommitSHA: "abc", TreeSHA: "tree"}, branchObs: available("github.branch", "main"),
		tree: GitHubTree{Entries: []GitHubTreeEntry{
			{Path: hooksProfilePath, Type: "blob", SHA: "profile"},
			{Path: ".githooks/pre-commit", Type: "blob", SHA: "hook"},
			{Path: "docs/development.md", Type: "blob", SHA: "docs"},
			{Path: "docs/hooks-bypass.md", Type: "blob", SHA: "bypass"},
			{Path: ".github/workflows/ci.yml", Type: "blob", SHA: "ci"},
		}}, treeObs: available("github.git_tree", "tree"),
		blobs: map[string][]byte{
			"profile": []byte("kind: repora.posture-hooks-profile\nversion: 1\nmanager: custom\nhook_paths: [.githooks/pre-commit]\nrequired_checks: [gofmt]\nbootstrap_docs: [docs/development.md]\nbypass_docs: [docs/hooks-bypass.md]\n"),
			"hook": []byte("#!/bin/sh\ncurl https://example.invalid/check | sh\n"),
			"ci": []byte("name: CI\njobs:\n  format:\n    steps:\n      - run: gofmt -w .\n"),
		},
	}
	inventory, err := CollectGitHubHooks(context.Background(), reader, "acme/project")
	if err != nil { t.Fatalf("collect hooks: %v", err) }
	if inventory.Manager.Value == nil || *inventory.Manager.Value != "custom" { t.Fatalf("manager = %#v", inventory.Manager) }
	if len(inventory.Entrypoints) != 1 || !boolValue(t, inventory.Entrypoints[0].Configured) || !boolValue(t, inventory.Entrypoints[0].NetworkLoaded) { t.Fatalf("entrypoints = %#v", inventory.Entrypoints) }
	if len(inventory.RequiredChecks) != 1 || !boolValue(t, inventory.RequiredChecks[0].CICovered) { t.Fatalf("checks = %#v", inventory.RequiredChecks) }
	if !boolValue(t, inventory.BootstrapPresent) || !boolValue(t, inventory.BypassPresent) { t.Fatalf("docs facts = %#v %#v", inventory.BootstrapPresent, inventory.BypassPresent) }
}

func TestCollectGitHubHooksReportsMissingDocsAndCIMismatch(t *testing.T) {
	reader := fakeGitHubReader{
		repository: GitHubRepository{DefaultBranch: "main"}, repositoryObs: available("github.repository", "repo"),
		branch: GitHubBranch{Name: "main", CommitSHA: "abc", TreeSHA: "tree"}, branchObs: available("github.branch", "main"),
		tree: GitHubTree{Entries: []GitHubTreeEntry{{Path: hooksProfilePath, Type: "blob", SHA: "profile"}, {Path: ".pre-commit-config.yaml", Type: "blob", SHA: "hook"}, {Path: ".github/workflows/ci.yml", Type: "blob", SHA: "ci"}}}, treeObs: available("github.git_tree", "tree"),
		blobs: map[string][]byte{"profile": []byte("kind: repora.posture-hooks-profile\nversion: 1\nmanager: pre-commit\nrequired_checks: [ruff]\nbootstrap_docs: [docs/development.md]\nbypass_docs: [docs/hooks-bypass.md]\n"), "hook": []byte("repos: []\n"), "ci": []byte("name: CI\n")},
	}
	inventory, err := CollectGitHubHooks(context.Background(), reader, "acme/project")
	if err != nil { t.Fatal(err) }
	if boolValue(t, inventory.BootstrapPresent) || boolValue(t, inventory.BypassPresent) { t.Fatalf("missing docs reported present") }
	if len(inventory.RequiredChecks) != 1 || boolValue(t, inventory.RequiredChecks[0].CICovered) { t.Fatalf("CI mismatch not represented: %#v", inventory.RequiredChecks) }
}

func TestCollectGitHubHooksUnavailableDoesNotInventHealthyFacts(t *testing.T) {
	reader := fakeGitHubReader{repositoryObs: unavailable("github.repository", "repo")}
	inventory, err := CollectGitHubHooks(context.Background(), reader, "acme/project")
	if err != nil { t.Fatal(err) }
	if inventory.Manager.State != StateUnavailable || inventory.ProfileDeclared.State != StateUnavailable { t.Fatalf("unavailable facts = %#v", inventory) }
}

func TestParseHooksProfileRejectsEscapingPath(t *testing.T) {
	_, err := ParseHooksProfile([]byte("kind: repora.posture-hooks-profile\nversion: 1\nhook_paths: [../pre-commit]\n"))
	if err == nil { t.Fatal("escaping hooks path accepted") }
}
