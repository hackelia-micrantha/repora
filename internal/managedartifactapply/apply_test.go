package managedartifactapply

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/journal"
	"repoctl/internal/managedartifact"
)

type fakeObserver struct{}

func (fakeObserver) ObserveREADME(config.Repo) (managedartifact.READMEObservation, error) {
	return managedartifact.READMEObservation{}, nil
}

type fakePreparer struct {
	events   *[]string
	prepared []managedartifact.PreparedCommit
	err      error
}

func (f fakePreparer) Prepare(config.Spec, managedartifact.Plan, managedartifact.READMEObserver) ([]managedartifact.PreparedCommit, error) {
	*f.events = append(*f.events, "prepare")
	return f.prepared, f.err
}

type fakePusher struct {
	events *[]string
	pushes []managedartifact.PushResult
	err    error
}

func (f fakePusher) Push(config.Spec, managedartifact.Plan, []managedartifact.PreparedCommit, managedartifact.READMEObserver) ([]managedartifact.PushResult, error) {
	*f.events = append(*f.events, "push")
	return f.pushes, f.err
}

type fakeJournalWriter struct {
	events     *[]string
	failIntent bool
	failResult bool
	records    []journal.ManagedArtifactRecord
}

func (w *fakeJournalWriter) WriteManagedArtifact(record journal.ManagedArtifactRecord) (string, error) {
	*w.events = append(*w.events, "journal:"+string(record.Phase))
	w.records = append(w.records, record)
	ref := ".repora/journal/test--" + strings.ToLower(string(record.Phase)) + ".json"
	if record.Phase == journal.PhaseIntent && w.failIntent {
		return "", errors.New("intent disk failure")
	}
	if record.Phase == journal.PhaseResult && w.failResult {
		return ref, errors.New("result sync failure")
	}
	return ref, nil
}

func TestExecuteOrdersIntentBeforePreparationAndPush(t *testing.T) {
	plan := applyFixturePlan(t, 1)
	events := []string{}
	prepared := applyFixturePrepared(plan)
	pushes := applyFixturePushes(plan, prepared, -1)
	writer := &fakeJournalWriter{events: &events}

	result, err := Execute(config.Spec{}, plan, fakeObserver{}, fakePreparer{events: &events, prepared: prepared}, fakePusher{events: &events, pushes: pushes}, Audit{ExecutionID: "run-order", Writer: writer})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"journal:INTENT", "prepare", "push", "journal:RESULT"}
	if fmt.Sprint(events) != fmt.Sprint(want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
	if result.Outcome != journal.OutcomeApplied || !result.Repositories[0].Pushed || result.Journal.Intent == "" || result.Journal.Result == "" {
		t.Fatalf("result = %+v", result)
	}
}

func TestExecuteIntentFailureBlocksAllMutation(t *testing.T) {
	plan := applyFixturePlan(t, 1)
	events := []string{}
	writer := &fakeJournalWriter{events: &events, failIntent: true}
	_, err := Execute(config.Spec{}, plan, fakeObserver{}, fakePreparer{events: &events}, fakePusher{events: &events}, Audit{ExecutionID: "run-intent-fail", Writer: writer})
	if err == nil || !strings.Contains(err.Error(), "intent disk failure") {
		t.Fatalf("error = %v", err)
	}
	if fmt.Sprint(events) != fmt.Sprint([]string{"journal:INTENT"}) {
		t.Fatalf("events = %v, want intent only", events)
	}
}

func TestExecutePersistsStalePreparationResult(t *testing.T) {
	plan := applyFixturePlan(t, 1)
	events := []string{}
	writer := &fakeJournalWriter{events: &events}
	stale := fmt.Errorf("preflight: %w", managedartifact.ErrStale)
	result, err := Execute(config.Spec{}, plan, fakeObserver{}, fakePreparer{events: &events, err: stale}, fakePusher{events: &events}, Audit{ExecutionID: "run-stale", Writer: writer})
	if !errors.Is(err, managedartifact.ErrStale) {
		t.Fatalf("error = %v, want ErrStale", err)
	}
	if fmt.Sprint(events) != fmt.Sprint([]string{"journal:INTENT", "prepare", "journal:RESULT"}) {
		t.Fatalf("events = %v", events)
	}
	if result.Outcome != journal.OutcomeStale || result.FailureStage != "STALE" || result.Journal.Result == "" {
		t.Fatalf("result = %+v", result)
	}
}

func TestExecuteProjectsPartialPushAndPersistsResult(t *testing.T) {
	plan := applyFixturePlan(t, 2)
	events := []string{}
	prepared := applyFixturePrepared(plan)
	pushes := applyFixturePushes(plan, prepared, 1)
	writer := &fakeJournalWriter{events: &events}

	result, err := Execute(config.Spec{}, plan, fakeObserver{}, fakePreparer{events: &events, prepared: prepared}, fakePusher{events: &events, pushes: pushes, err: errors.New("second push rejected")}, Audit{ExecutionID: "run-partial", Writer: writer})
	if err == nil || !strings.Contains(err.Error(), "second push rejected") {
		t.Fatalf("error = %v", err)
	}
	if result.Outcome != journal.OutcomeFailed || result.FailureStage != "PUSH" || len(result.Repositories) != 2 {
		t.Fatalf("result = %+v", result)
	}
	if !result.Repositories[0].Pushed || result.Repositories[0].Outcome != journal.OutcomeApplied {
		t.Fatalf("first result = %+v", result.Repositories[0])
	}
	if result.Repositories[1].Pushed || result.Repositories[1].Outcome != journal.OutcomeFailed {
		t.Fatalf("second result = %+v", result.Repositories[1])
	}
	if result.Journal.Result == "" {
		t.Fatalf("result journal reference missing: %+v", result)
	}
}

func TestExecuteResultJournalFailureDoesNotHideAppliedOutcome(t *testing.T) {
	plan := applyFixturePlan(t, 1)
	events := []string{}
	prepared := applyFixturePrepared(plan)
	pushes := applyFixturePushes(plan, prepared, -1)
	writer := &fakeJournalWriter{events: &events, failResult: true}

	result, err := Execute(config.Spec{}, plan, fakeObserver{}, fakePreparer{events: &events, prepared: prepared}, fakePusher{events: &events, pushes: pushes}, Audit{ExecutionID: "run-result-fail", Writer: writer})
	if err == nil || !strings.Contains(err.Error(), "result sync failure") {
		t.Fatalf("error = %v", err)
	}
	if result.Outcome != journal.OutcomeApplied || !result.Repositories[0].Pushed || result.Journal.Result == "" {
		t.Fatalf("applied result was hidden: %+v", result)
	}
}

func applyFixturePlan(t *testing.T, count int) managedartifact.Plan {
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
		t.Fatal(err)
	}
	return plan
}

func applyFixturePrepared(plan managedartifact.Plan) []managedartifact.PreparedCommit {
	prepared := make([]managedartifact.PreparedCommit, 0, len(plan.Repositories))
	for i, repo := range plan.Repositories {
		prepared = append(prepared, managedartifact.PreparedCommit{UID: repo.UID, ID: repo.ID, BaseOID: repo.BaseOID, TreeOID: strings.Repeat(string(rune('a'+i)), 40), CommitOID: strings.Repeat(string(rune('d'+i)), 40)})
	}
	return prepared
}

func applyFixturePushes(plan managedartifact.Plan, prepared []managedartifact.PreparedCommit, failIndex int) []managedartifact.PushResult {
	pushes := make([]managedartifact.PushResult, 0, len(plan.Repositories))
	for i, repo := range plan.Repositories {
		pushed := failIndex < 0 || i < failIndex
		if failIndex >= 0 && i > failIndex {
			break
		}
		pushes = append(pushes, managedartifact.PushResult{UID: repo.UID, ID: repo.ID, Branch: repo.Target.Branch, BaseOID: repo.BaseOID, CommitOID: prepared[i].CommitOID, Pushed: pushed})
	}
	return pushes
}
