package journal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"repoctl/internal/managedartifact"
)

func TestManagedArtifactIntentRoundTripDoesNotSerializeDesiredContent(t *testing.T) {
	plan := managedJournalPlan(t, 1)
	record, err := ManagedArtifactIntent("run-test", plan)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := record.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "desired-a") {
		t.Fatalf("journal intent leaked desired README content:\n%s", encoded)
	}
	parsed, err := ParseManagedArtifact(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Phase != PhaseIntent || parsed.Outcome != OutcomePlanned || parsed.Repositories[0].DesiredSHA256 == "" {
		t.Fatalf("parsed intent = %+v", parsed)
	}
}

func TestManagedArtifactResultPreservesPartialPushSuccess(t *testing.T) {
	plan := managedJournalPlan(t, 2)
	prepared := []managedartifact.PreparedCommit{
		{UID: "repo.a", ID: "a", BaseOID: strings.Repeat("1", 40), TreeOID: strings.Repeat("a", 40), CommitOID: strings.Repeat("b", 40)},
		{UID: "repo.b", ID: "b", BaseOID: strings.Repeat("2", 40), TreeOID: strings.Repeat("c", 40), CommitOID: strings.Repeat("d", 40)},
	}
	pushes := []managedartifact.PushResult{
		{UID: "repo.a", ID: "a", Branch: "main", BaseOID: strings.Repeat("1", 40), CommitOID: strings.Repeat("b", 40), Pushed: true},
		{UID: "repo.b", ID: "b", Branch: "main", BaseOID: strings.Repeat("2", 40), CommitOID: strings.Repeat("d", 40), Pushed: false},
	}
	record, err := ManagedArtifactResult("run-partial", plan, prepared, pushes, errors.New("push rejected"))
	if err != nil {
		t.Fatal(err)
	}
	if record.Outcome != OutcomeFailed || record.FailureStage != "PUSH" {
		t.Fatalf("overall result = %+v", record)
	}
	if record.Repositories[0].Outcome != OutcomeApplied || !record.Repositories[0].Pushed {
		t.Fatalf("first repo = %+v", record.Repositories[0])
	}
	if record.Repositories[1].Outcome != OutcomeFailed || record.Repositories[1].Pushed {
		t.Fatalf("second repo = %+v", record.Repositories[1])
	}
}

func TestManagedArtifactResultUsesTypedStaleError(t *testing.T) {
	plan := managedJournalPlan(t, 1)
	err := fmt.Errorf("preflight: %w", managedartifact.ErrStale)
	record, buildErr := ManagedArtifactResult("run-stale", plan, nil, nil, err)
	if buildErr != nil {
		t.Fatal(buildErr)
	}
	if record.Outcome != OutcomeStale || record.FailureStage != "STALE" || record.Repositories[0].Outcome != OutcomeStale {
		t.Fatalf("stale result = %+v", record)
	}
}

func TestWriterPersistsManagedArtifactIntentAndResultWithoutOverwrite(t *testing.T) {
	root := t.TempDir()
	plan := managedJournalPlan(t, 1)
	intent, err := ManagedArtifactIntent("run-write", plan)
	if err != nil {
		t.Fatal(err)
	}
	writer := Writer{Root: root}
	intentRef, err := writer.WriteManagedArtifact(intent)
	if err != nil {
		t.Fatal(err)
	}
	if intentRef != ".repora/journal/managed-artifact--run-write--intent.json" {
		t.Fatalf("intent ref = %q", intentRef)
	}
	if _, err := writer.WriteManagedArtifact(intent); !errors.Is(err, ErrRecordExists) {
		t.Fatalf("second intent write error = %v, want ErrRecordExists", err)
	}

	prepared := []managedartifact.PreparedCommit{{UID: "repo.a", ID: "a", BaseOID: strings.Repeat("1", 40), TreeOID: strings.Repeat("a", 40), CommitOID: strings.Repeat("b", 40)}}
	pushes := []managedartifact.PushResult{{UID: "repo.a", ID: "a", Branch: "main", BaseOID: strings.Repeat("1", 40), CommitOID: strings.Repeat("b", 40), Pushed: true}}
	result, err := ManagedArtifactResult("run-write", plan, prepared, pushes, nil)
	if err != nil {
		t.Fatal(err)
	}
	resultRef, err := writer.WriteManagedArtifact(result)
	if err != nil {
		t.Fatal(err)
	}
	if resultRef != ".repora/journal/managed-artifact--run-write--result.json" {
		t.Fatalf("result ref = %q", resultRef)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(resultRef)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseManagedArtifact(data); err != nil {
		t.Fatalf("parse persisted result: %v", err)
	}
}

func managedJournalPlan(t *testing.T, count int) managedartifact.Plan {
	t.Helper()
	plan := managedartifact.Plan{Kind: managedartifact.PlanKind, Version: managedartifact.PlanVersion, Repositories: []managedartifact.RepositoryPlan{}}
	for i := 0; i < count; i++ {
		id := string(rune('a' + i))
		desired := "desired-" + id + "\n"
		diff, err := managedartifact.ReviewDiff(false, nil, []byte(desired))
		if err != nil {
			t.Fatal(err)
		}
		present := false
		plan.Repositories = append(plan.Repositories, managedartifact.RepositoryPlan{
			UID:     "repo." + id,
			ID:      id,
			Target:  managedartifact.Target{Provider: "gitlab", Path: "example/" + id, Branch: "main"},
			BaseOID: strings.Repeat(string(rune('1'+i)), 40),
			Actions: []managedartifact.Action{{
				Type:           managedartifact.ActionWriteREADME,
				Path:           managedartifact.READMEPath,
				Observed:       managedartifact.ObservedState{Present: &present},
				Desired:        managedartifact.DesiredState{Mode: "100644", SHA256: managedartifact.DigestSHA256([]byte(desired)), Content: &desired},
				TemplateSHA256: strings.Repeat("c", 64),
				Diff:           diff,
			}},
		})
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("fixture plan: %v", err)
	}
	return plan
}
