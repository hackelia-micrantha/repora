package apply

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"repoctl/internal/journal"
	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

func TestExecuteArtifactAuditedWritesIntentBeforeMutationAndResultAfter(t *testing.T) {
	git := &fakeGit{}
	writer := &recordingJournalWriter{}
	repo := testRepo()
	artifact := auditedArtifact(repo, false)

	got, err := ExecuteArtifactAudited(repo, status.Result{State: status.StateBehind}, artifact, git, false, false, Audit{
		ExecutionID: "run-apply",
		Writer:      writer,
	})
	if err != nil {
		t.Fatalf("ExecuteArtifactAudited returned error: %v", err)
	}
	if len(writer.records) != 2 || writer.records[0].Phase != journal.PhaseIntent || writer.records[1].Phase != journal.PhaseResult {
		t.Fatalf("records = %#v, want INTENT then RESULT", writer.records)
	}
	if writer.records[1].Actions[0].Outcome != journal.OutcomeApplied || writer.records[1].Actions[0].After != testOID {
		t.Fatalf("result record = %#v, want applied evidence", writer.records[1])
	}
	if len(git.pushBranchCalls) != 1 {
		t.Fatalf("push calls = %#v, want one mutation", git.pushBranchCalls)
	}
	if got.Journal == nil || got.Journal.ExecutionID != "run-apply" || got.Journal.Intent == "" || got.Journal.Result == "" {
		t.Fatalf("journal references = %#v, want complete references", got.Journal)
	}
}

func TestExecuteArtifactAuditedFailsClosedWhenIntentWriteFails(t *testing.T) {
	git := &fakeGit{}
	writer := &recordingJournalWriter{failAt: 1}
	repo := testRepo()

	got, err := ExecuteArtifactAudited(repo, status.Result{State: status.StateBehind}, auditedArtifact(repo, false), git, false, false, Audit{
		ExecutionID: "run-intent-fail",
		Writer:      writer,
	})
	if err == nil || !strings.Contains(err.Error(), "persist journal intent") {
		t.Fatalf("error = %v, want intent persistence failure", err)
	}
	assertNoMutation(t, git)
	if got.Journal == nil || got.Journal.Intent == "" || got.Journal.Result != "" {
		t.Fatalf("journal references = %#v, want attempted intent reference only", got.Journal)
	}
}

func TestExecuteArtifactAuditedPersistsRuntimeFailure(t *testing.T) {
	git := &fakeGit{pushBranchErr: errors.New("remote rejected update")}
	writer := &recordingJournalWriter{}
	repo := testRepo()

	got, err := ExecuteArtifactAudited(repo, status.Result{State: status.StateBehind}, auditedArtifact(repo, false), git, false, false, Audit{
		ExecutionID: "run-push-fail",
		Writer:      writer,
	})
	if err == nil || !strings.Contains(err.Error(), "remote rejected update") {
		t.Fatalf("error = %v, want mutation failure", err)
	}
	if len(writer.records) != 2 || writer.records[1].Actions[0].Outcome != journal.OutcomeFailed {
		t.Fatalf("records = %#v, want durable failed result", writer.records)
	}
	if got.Journal == nil || got.Journal.Result == "" {
		t.Fatalf("journal references = %#v, want result reference", got.Journal)
	}
}

func TestExecuteArtifactAuditedReturnsResultWriteFailureAfterMutation(t *testing.T) {
	git := &fakeGit{}
	writer := &recordingJournalWriter{failAt: 2}
	repo := testRepo()

	got, err := ExecuteArtifactAudited(repo, status.Result{State: status.StateBehind}, auditedArtifact(repo, false), git, false, false, Audit{
		ExecutionID: "run-result-fail",
		Writer:      writer,
	})
	if err == nil || !strings.Contains(err.Error(), "persist journal result") {
		t.Fatalf("error = %v, want result persistence failure", err)
	}
	if len(git.pushBranchCalls) != 1 {
		t.Fatalf("push calls = %#v, want completed mutation", git.pushBranchCalls)
	}
	if !got.Applied || got.Journal == nil || got.Journal.Intent == "" || got.Journal.Result == "" {
		t.Fatalf("result = %#v, want applied state and attempted journal references", got)
	}
}

func TestExecuteArtifactAuditedDryRunWritesValidatedResultWithoutMutation(t *testing.T) {
	git := &fakeGit{}
	writer := &recordingJournalWriter{}
	repo := testRepo()

	got, err := ExecuteArtifactAudited(repo, status.Result{State: status.StateBehind}, auditedArtifact(repo, false), git, false, true, Audit{
		ExecutionID: "run-dry",
		Writer:      writer,
	})
	if err != nil {
		t.Fatalf("ExecuteArtifactAudited returned error: %v", err)
	}
	assertNoMutation(t, git)
	if len(writer.records) != 2 || writer.records[0].Mode != journal.ModeDryRun || writer.records[1].Actions[0].Outcome != journal.OutcomeValidated {
		t.Fatalf("records = %#v, want dry-run intent and validated result", writer.records)
	}
	if !got.DryRun || got.Journal == nil || got.Journal.Result == "" {
		t.Fatalf("result = %#v, want audited dry-run", got)
	}
}

func TestExecuteArtifactAuditedRejectsMissingAuditConfiguration(t *testing.T) {
	repo := testRepo()
	artifact := auditedArtifact(repo, false)
	if _, err := ExecuteArtifactAudited(repo, status.Result{State: status.StateBehind}, artifact, &fakeGit{}, false, false, Audit{}); err == nil || !strings.Contains(err.Error(), "execution ID") {
		t.Fatalf("error = %v, want execution ID validation", err)
	}
	if _, err := ExecuteArtifactAudited(repo, status.Result{State: status.StateBehind}, artifact, &fakeGit{}, false, false, Audit{ExecutionID: "run"}); err == nil || !strings.Contains(err.Error(), "writer") {
		t.Fatalf("error = %v, want writer validation", err)
	}
}

func auditedArtifact(repo config.Repo, force bool) planartifact.Artifact {
	return planartifact.FromPlans(plan.ReconciliationPlan{
		ID:  repo.ID,
		UID: repo.DurableID(),
		Actions: []plan.PlannedAction{{
			Type:              plan.ActionPushBranch,
			Source:            plan.Remote{Provider: repo.Canonical.Provider, Name: "canonical", Branch: "main"},
			Target:            plan.Remote{Provider: repo.Mirrors[0].Provider, Name: "mirror", Branch: "main"},
			ExpectedSource:    testOID,
			ExpectedOldTarget: testOID,
			Force:             force,
			Reason:            "mirror is behind",
		}},
	})
}

type recordingJournalWriter struct {
	records []journal.Record
	writes  int
	failAt  int
}

func (w *recordingJournalWriter) Write(record journal.Record) (string, error) {
	w.writes++
	w.records = append(w.records, record)
	ref := fmt.Sprintf(".repora/journal/%s--%s--%s.json", record.Repository.UID, record.ExecutionID, strings.ToLower(string(record.Phase)))
	if w.failAt == w.writes {
		return ref, errors.New("journal unavailable")
	}
	return ref, nil
}
