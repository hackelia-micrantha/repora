package main

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"repoctl/internal/posture"
)

func validCIEnvironmentInventory(fullName string) posture.CIEnvironmentInventory {
	return posture.CIEnvironmentInventory{
		Kind:                  posture.CIEnvironmentInventoryKind,
		Version:               posture.CIEnvironmentInventoryVersion,
		Repository:            posture.RepositoryIdentity{Provider: "github", FullName: fullName},
		DefaultBranch:         posture.Observed("main"),
		DefaultCommit:         posture.Observed("abc1234"),
		ProfileDeclared:       posture.Observed(false),
		DeclaredApplicability: posture.Unknown[string](),
		FlakePresent:          posture.Observed(true),
		FlakeLockPresent:      posture.Observed(true),
		WorkflowsState:        posture.StateObserved,
		Workflows:             []posture.CIEnvironmentWorkflowFact{},
		ExternalInputs:        []posture.CIExternalInputFact{},
		Evidence:              []posture.Evidence{},
	}
}

func TestPostureCIEnvironmentCommandEmitsVersionedJSON(t *testing.T) {
	old := collectGitHubCIEnvironmentPosture
	t.Cleanup(func() { collectGitHubCIEnvironmentPosture = old })
	t.Setenv("GITHUB_TOKEN", "ci-env-token")
	var gotRepo, gotToken string
	collectGitHubCIEnvironmentPosture = func(_ context.Context, fullName, token string) (posture.CIEnvironmentInventory, error) {
		gotRepo, gotToken = fullName, token
		return validCIEnvironmentInventory(fullName), nil
	}
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"posture", "ci-environment", "acme/project"})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if gotRepo != "acme/project" || gotToken != "ci-env-token" {
		t.Fatalf("collector inputs repo=%q token=%q", gotRepo, gotToken)
	}
	var decoded posture.CIEnvironmentInventory
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	if decoded.Kind != posture.CIEnvironmentInventoryKind || decoded.Version != posture.CIEnvironmentInventoryVersion {
		t.Fatalf("CI environment envelope = %#v", decoded)
	}
}
