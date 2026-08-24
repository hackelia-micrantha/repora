package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"repoctl/internal/posture"
	"repoctl/internal/posturepolicy"
)

func TestPostureConvergeAndReportCommandsAreDeterministic(t *testing.T) {
	dir := t.TempDir()
	fullName := "acme/project"

	inventoryData, err := validPostureInventory(fullName).Marshal()
	if err != nil {
		t.Fatal(err)
	}
	docsData, err := validDocumentationInventory(fullName).Marshal()
	if err != nil {
		t.Fatal(err)
	}
	hooksData, err := validHooksInventory(fullName).Marshal()
	if err != nil {
		t.Fatal(err)
	}
	commitsData, err := validCommitInventory(fullName).Marshal()
	if err != nil {
		t.Fatal(err)
	}
	mirrors := validMirrorInventoryForConvergeCommand(fullName)
	mirrorsData, err := mirrors.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	inventoryPath := writePostureTestFile(t, dir, "inventory.json", inventoryData)
	docsPath := writePostureTestFile(t, dir, "docs.json", docsData)
	hooksPath := writePostureTestFile(t, dir, "hooks.json", hooksData)
	commitsPath := writePostureTestFile(t, dir, "commits.json", commitsData)
	mirrorsPath := writePostureTestFile(t, dir, "mirrors.json", mirrorsData)

	convergeArgs := []string{
		"posture", "converge",
		"--inventory", inventoryPath,
		"--docs", docsPath,
		"--hooks", hooksPath,
		"--commits", commitsPath,
		"--mirrors", mirrorsPath,
		"--repo-uid", "repo.project",
	}

	var firstFacts bytes.Buffer
	if code := withStdout(t, &firstFacts, func() int { return run(convergeArgs) }); code != 0 {
		t.Fatalf("first converge returned %d", code)
	}
	var secondFacts bytes.Buffer
	if code := withStdout(t, &secondFacts, func() int { return run(convergeArgs) }); code != 0 {
		t.Fatalf("second converge returned %d", code)
	}
	if !bytes.Equal(firstFacts.Bytes(), secondFacts.Bytes()) {
		t.Fatalf("repeated converge output differs:\nfirst:\n%s\nsecond:\n%s", firstFacts.Bytes(), secondFacts.Bytes())
	}

	inputs, err := posturepolicy.ParseInputs(firstFacts.Bytes())
	if err != nil {
		t.Fatalf("parse converged facts: %v", err)
	}
	if inputs.Repository != fullName {
		t.Fatalf("repository = %q, want %q", inputs.Repository, fullName)
	}
	for _, fact := range []string{
		"repository.default_branch",
		"documentation.readme_present",
		"hooks.manager",
		"commits.observed_count",
		"mirrors.repo_uid",
	} {
		if _, ok := inputs.Facts[fact]; !ok {
			t.Fatalf("converged facts missing %q", fact)
		}
	}

	factsPath := writePostureTestFile(t, dir, "facts.json", firstFacts.Bytes())
	profile := posturepolicy.Profile{
		Kind:    posturepolicy.ProfileKind,
		Version: posturepolicy.ProfileVersion,
		ID:      "baseline",
		Rules: []posturepolicy.Rule{{
			ID:          "default-branch-protected",
			Area:        "repository",
			Fact:        "repository.default_branch_protected",
			Operator:    posturepolicy.OperatorEquals,
			Expected:    json.RawMessage(`true`),
			Severity:    posturepolicy.SeverityHigh,
			Title:       "Default branch is protected",
			Remediation: []string{},
		}},
		Exceptions: []posturepolicy.Exception{},
	}
	profileData, err := profile.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	profilePath := writePostureTestFile(t, dir, "policy.json", profileData)
	reportArgs := []string{
		"posture", "report",
		"--profile", profilePath,
		"--facts", factsPath,
		"--as-of", "2026-08-23",
		"--format", "json",
	}

	var firstReport bytes.Buffer
	if code := withStdout(t, &firstReport, func() int { return run(reportArgs) }); code != 0 {
		t.Fatalf("first report returned %d", code)
	}
	var secondReport bytes.Buffer
	if code := withStdout(t, &secondReport, func() int { return run(reportArgs) }); code != 0 {
		t.Fatalf("second report returned %d", code)
	}
	if !bytes.Equal(firstReport.Bytes(), secondReport.Bytes()) {
		t.Fatalf("repeated report output differs:\nfirst:\n%s\nsecond:\n%s", firstReport.Bytes(), secondReport.Bytes())
	}

	var report posturepolicy.Report
	if err := json.Unmarshal(firstReport.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if err := report.Validate(); err != nil {
		t.Fatalf("validate report: %v", err)
	}
}

func writePostureTestFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func validMirrorInventoryForConvergeCommand(githubPath string) posture.MirrorInventory {
	inventory := posture.NewMirrorInventory()
	inventory.Repos = append(inventory.Repos, posture.MirrorRepositoryFacts{
		ID:        "project",
		UID:       "repo.project",
		Mode:      posture.Observed("mirror"),
		Direction: posture.Observed("canonical_to_mirror"),
		Canonical: posture.MirrorCanonicalFacts{
			Identity:                   posture.MirrorEndpointIdentity{Provider: "gitlab", Path: "acme/project"},
			DefaultBranch:              posture.Observed("main"),
			Commit:                     posture.Observed("abc1234"),
			Visibility:                 posture.Observed("private"),
			CurrentActorPushPermission: posture.Observed(true),
		},
		Mirrors: []posture.MirrorTargetFacts{{
			Identity:                   posture.MirrorEndpointIdentity{Provider: "github", Path: githubPath},
			CacheRemote:                posture.Observed("mirror-0"),
			DefaultBranch:              posture.Observed("main"),
			DefaultBranchDrift:         posture.Observed(false),
			Commit:                     posture.Observed("abc1234"),
			Divergence:                 posture.Observed("in_sync"),
			Ahead:                      posture.Observed(0),
			Behind:                     posture.Observed(0),
			Visibility:                 posture.Observed("public"),
			CurrentActorPushPermission: posture.Observed(true),
			TagDrift:                   posture.Unknown[bool](),
			ReleaseDrift:               posture.Unknown[bool](),
		}},
		Evidence: []posture.Evidence{},
	})
	return inventory
}
