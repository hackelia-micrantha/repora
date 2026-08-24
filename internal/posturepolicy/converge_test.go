package posturepolicy

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"repoctl/internal/posture"
)

func TestConvergeArtifactsRepeatedRunDeterminism(t *testing.T) {
	inventory := posture.NewInventory("acme/project")
	inventory.RepositoryFacts = validRepositoryFactsForAdapterTest()
	inventory.WorkflowsState = posture.StateObserved
	inventoryData, err := inventory.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	artifacts := ArtifactSet{Inventory: inventoryData}
	firstInputs, err := ConvergeArtifacts(artifacts)
	if err != nil {
		t.Fatal(err)
	}
	secondInputs, err := ConvergeArtifacts(artifacts)
	if err != nil {
		t.Fatal(err)
	}
	firstFacts, err := firstInputs.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	secondFacts, err := secondInputs.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstFacts, secondFacts) {
		t.Fatalf("repeated convergence differs:\nfirst:\n%s\nsecond:\n%s", firstFacts, secondFacts)
	}

	profile := Profile{
		Kind:    ProfileKind,
		Version: ProfileVersion,
		ID:      "baseline",
		Rules: []Rule{{
			ID:          "default-branch-protected",
			Area:        "repository",
			Fact:        "repository.default_branch_protected",
			Operator:    OperatorEquals,
			Expected:    json.RawMessage(`true`),
			Severity:    SeverityHigh,
			Title:       "Default branch is protected",
			Remediation: []string{"Enable default-branch protection."},
		}},
		Exceptions: []Exception{},
	}
	asOf := time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC)
	firstReport, err := Evaluate(profile, firstInputs, asOf)
	if err != nil {
		t.Fatal(err)
	}
	secondReport, err := Evaluate(profile, secondInputs, asOf)
	if err != nil {
		t.Fatal(err)
	}
	firstReportData, err := firstReport.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	secondReportData, err := secondReport.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstReportData, secondReportData) {
		t.Fatalf("repeated report differs:\nfirst:\n%s\nsecond:\n%s", firstReportData, secondReportData)
	}
}

func TestConvergeArtifactsRejectsUnknownFields(t *testing.T) {
	inventory := posture.NewInventory("acme/project")
	inventory.RepositoryFacts = validRepositoryFactsForAdapterTest()
	inventory.WorkflowsState = posture.StateObserved
	data, err := inventory.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.Replace(data, []byte(`"version": 1,`), []byte(`"version": 1, "unexpected": true,`), 1)
	if _, err := ConvergeArtifacts(ArtifactSet{Inventory: data}); err == nil {
		t.Fatal("expected unknown field to be rejected")
	}
}

func TestConvergeArtifactsRejectsMirrorRepositoryMismatch(t *testing.T) {
	inventory := posture.NewInventory("acme/project")
	inventory.RepositoryFacts = validRepositoryFactsForAdapterTest()
	inventory.WorkflowsState = posture.StateObserved
	inventoryData, err := inventory.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	mirrors := posture.NewMirrorInventory()
	mirrors.Repos = append(mirrors.Repos, validMirrorRepositoryForConvergeTest("other/project"))
	mirrorData, err := mirrors.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ConvergeArtifacts(ArtifactSet{
		Inventory:     inventoryData,
		Mirrors:       mirrorData,
		MirrorRepoUID: "repo.project",
	}); err == nil {
		t.Fatal("expected mirror repository mismatch")
	}
}

func validMirrorRepositoryForConvergeTest(githubPath string) posture.MirrorRepositoryFacts {
	return posture.MirrorRepositoryFacts{
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
	}
}
