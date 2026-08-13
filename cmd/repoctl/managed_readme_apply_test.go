package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/journal"
	"repoctl/internal/managedartifact"
	"repoctl/internal/managedartifactapply"
)

func TestApplyREADMEDryRunPrintsReviewedPlan(t *testing.T) {
	configPath := writeManagedPlanConfig(t)
	plan := managedPlanFixture(t)
	planPath := writeManagedPlanFile(t, plan)
	stubManagedPreflight(t, nil)

	var stdout bytes.Buffer
	code := captureManagedStdout(t, &stdout, func() int {
		return run([]string{"apply-readme", "-f", configPath, "--plan-file", planPath, "--dry-run"})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), plan.Repositories[0].Actions[0].Diff) {
		t.Fatalf("stdout missing reviewed diff:\n%s", stdout.String())
	}
}

func TestApplyREADMEDryRunReturnsTwoForStalePlan(t *testing.T) {
	configPath := writeManagedPlanConfig(t)
	planPath := writeManagedPlanFile(t, managedPlanFixture(t))
	stubManagedPreflight(t, fmt.Errorf("%w: canonical HEAD changed", managedartifact.ErrStale))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := captureManagedStdout(t, &stdout, func() int {
		return captureManagedStderr(t, &stderr, func() int {
			return run([]string{"apply-readme", "-f", configPath, "--plan-file", planPath, "--dry-run"})
		})
	})
	if code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "managed artifact plan is stale") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestApplyREADMEDryRunReturnsOneForObservationFailure(t *testing.T) {
	configPath := writeManagedPlanConfig(t)
	planPath := writeManagedPlanFile(t, managedPlanFixture(t))
	stubManagedPreflight(t, errors.New("fetch unavailable"))

	var stderr bytes.Buffer
	code := captureManagedStderr(t, &stderr, func() int {
		return run([]string{"apply-readme", "-f", configPath, "--plan-file", planPath, "--dry-run"})
	})
	if code != 1 || !strings.Contains(stderr.String(), "fetch unavailable") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}

func TestApplyREADMERequiresPlanFileBeforeConfigIO(t *testing.T) {
	var stderr bytes.Buffer
	code := captureManagedStderr(t, &stderr, func() int {
		return run([]string{"apply-readme", "-f", "/does/not/exist"})
	})
	if code != 1 || !strings.Contains(stderr.String(), "requires --plan-file FILE") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}

func TestApplyREADMEDryRunRejectsJSONBeforeIO(t *testing.T) {
	var stderr bytes.Buffer
	code := captureManagedStderr(t, &stderr, func() int {
		return run([]string{"apply-readme", "--plan-file", "/does/not/exist", "--dry-run", "--json"})
	})
	if code != 1 || !strings.Contains(stderr.String(), "--json is supported for real apply only") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}

func TestApplyREADMERealApplyJSON(t *testing.T) {
	configPath := writeManagedPlanConfig(t)
	planPath := writeManagedPlanFile(t, managedPlanFixture(t))
	want := managedartifactapply.Result{
		Version:     managedartifactapply.ResultVersion,
		Kind:        managedartifactapply.ResultKind,
		ExecutionID: "run-cli",
		Outcome:     journal.OutcomeApplied,
		Repositories: []managedartifactapply.RepositoryResult{{
			UID: "repo.demo", ID: "demo", Branch: "main", BaseOID: strings.Repeat("1", 40), CommitOID: strings.Repeat("d", 40), Pushed: true, Outcome: journal.OutcomeApplied,
		}},
		Journal: managedartifactapply.JournalReferences{Intent: ".repora/journal/intent.json", Result: ".repora/journal/result.json"},
	}
	stubManagedApply(t, want, nil, nil)

	var stdout bytes.Buffer
	code := captureManagedStdout(t, &stdout, func() int {
		return run([]string{"apply-readme", "-f", configPath, "--plan-file", planPath, "--json"})
	})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	var got managedartifactapply.Result
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if got.Kind != managedartifactapply.ResultKind || len(got.Repositories) != 1 || !got.Repositories[0].Pushed {
		t.Fatalf("result = %+v", got)
	}
}

func TestApplyREADMERealApplyHumanOutputAndFailure(t *testing.T) {
	configPath := writeManagedPlanConfig(t)
	planPath := writeManagedPlanFile(t, managedPlanFixture(t))
	result := managedartifactapply.Result{
		Version:      managedartifactapply.ResultVersion,
		Kind:         managedartifactapply.ResultKind,
		ExecutionID:  "run-cli-fail",
		Outcome:      journal.OutcomeFailed,
		FailureStage: "PUSH",
		Repositories: []managedartifactapply.RepositoryResult{{
			UID: "repo.demo", ID: "demo", Branch: "main", BaseOID: strings.Repeat("1", 40), CommitOID: strings.Repeat("d", 40), Pushed: false, Outcome: journal.OutcomeFailed,
		}},
		Journal: managedartifactapply.JournalReferences{Intent: ".repora/journal/intent.json", Result: ".repora/journal/result.json"},
	}
	stubManagedApply(t, result, errors.New("lease rejected"), nil)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := captureManagedStdout(t, &stdout, func() int {
		return captureManagedStderr(t, &stderr, func() int {
			return run([]string{"apply-readme", "-f", configPath, "--plan-file", planPath})
		})
	})
	if code != 1 || !strings.Contains(stdout.String(), "demo (repo.demo) FAILED main") || !strings.Contains(stdout.String(), "journal intent:") || !strings.Contains(stderr.String(), "lease rejected") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestApplyREADMERealApplyStaleReturnsTwoAfterResultOutput(t *testing.T) {
	configPath := writeManagedPlanConfig(t)
	planPath := writeManagedPlanFile(t, managedPlanFixture(t))
	result := managedartifactapply.Result{Version: managedartifactapply.ResultVersion, Kind: managedartifactapply.ResultKind, ExecutionID: "run-stale", Outcome: journal.OutcomeStale, FailureStage: "STALE", Repositories: []managedartifactapply.RepositoryResult{{UID: "repo.demo", ID: "demo", Branch: "main", BaseOID: strings.Repeat("1", 40), Outcome: journal.OutcomeStale}}, Journal: managedartifactapply.JournalReferences{Intent: "intent", Result: "result"}}
	stubManagedApply(t, result, fmt.Errorf("%w: head changed", managedartifact.ErrStale), nil)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := captureManagedStdout(t, &stdout, func() int {
		return captureManagedStderr(t, &stderr, func() int {
			return run([]string{"apply-readme", "-f", configPath, "--plan-file", planPath})
		})
	})
	if code != 2 || !strings.Contains(stdout.String(), "STALE") || !strings.Contains(stderr.String(), "managed artifact plan is stale") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestApplyREADMEJournalInitializationFailureBlocksExecute(t *testing.T) {
	configPath := writeManagedPlanConfig(t)
	planPath := writeManagedPlanFile(t, managedPlanFixture(t))
	calls := 0
	stubManagedApplyWithCallCounter(t, &calls, managedartifactapply.Result{}, nil, errors.New("journal unavailable"))

	var stderr bytes.Buffer
	code := captureManagedStderr(t, &stderr, func() int {
		return run([]string{"apply-readme", "-f", configPath, "--plan-file", planPath})
	})
	if code != 1 || calls != 0 || !strings.Contains(stderr.String(), "journal unavailable") {
		t.Fatalf("code=%d calls=%d stderr=%q", code, calls, stderr.String())
	}
}

func TestApplyREADMEHelp(t *testing.T) {
	var stdout bytes.Buffer
	code := captureManagedStdout(t, &stdout, func() int {
		return run([]string{"apply-readme", "--help"})
	})
	if code != 0 || !strings.Contains(stdout.String(), "usage: repoctl apply-readme") {
		t.Fatalf("code=%d stdout=%q", code, stdout.String())
	}
}

func stubManagedPreflight(t *testing.T, preflightErr error) {
	t.Helper()
	oldPreflight := managedREADMEPreflight
	oldObserver := managedREADMEPreflightObserver
	managedREADMEPreflight = func(config.Spec, managedartifact.Plan, managedartifact.READMEObserver) error {
		return preflightErr
	}
	managedREADMEPreflightObserver = func() managedartifact.READMEObserver { return nil }
	t.Cleanup(func() {
		managedREADMEPreflight = oldPreflight
		managedREADMEPreflightObserver = oldObserver
	})
}

type noopManagedJournalWriter struct{}

func (noopManagedJournalWriter) WriteManagedArtifact(journal.ManagedArtifactRecord) (string, error) { return "", nil }

func stubManagedApply(t *testing.T, result managedartifactapply.Result, executeErr, journalErr error) {
	t.Helper()
	calls := 0
	stubManagedApplyWithCallCounter(t, &calls, result, executeErr, journalErr)
}

func stubManagedApplyWithCallCounter(t *testing.T, calls *int, result managedartifactapply.Result, executeErr, journalErr error) {
	t.Helper()
	oldExecute := managedREADMEApplyExecute
	oldContext := managedREADMEJournalContext
	oldPreparer := managedREADMECommitPreparer
	oldPusher := managedREADMEPusher
	oldObserver := managedREADMEPreflightObserver
	managedREADMEApplyExecute = func(config.Spec, managedartifact.Plan, managedartifact.READMEObserver, managedartifactapply.Preparer, managedartifactapply.Pusher, managedartifactapply.Audit) (managedartifactapply.Result, error) {
		*calls++
		return result, executeErr
	}
	managedREADMEJournalContext = func(string) (string, managedartifactapply.JournalWriter, error) {
		if journalErr != nil {
			return "", nil, journalErr
		}
		return "run-cli", noopManagedJournalWriter{}, nil
	}
	managedREADMECommitPreparer = func() managedartifactapply.Preparer { return nil }
	managedREADMEPusher = func() managedartifactapply.Pusher { return nil }
	managedREADMEPreflightObserver = func() managedartifact.READMEObserver { return nil }
	t.Cleanup(func() {
		managedREADMEApplyExecute = oldExecute
		managedREADMEJournalContext = oldContext
		managedREADMECommitPreparer = oldPreparer
		managedREADMEPusher = oldPusher
		managedREADMEPreflightObserver = oldObserver
	})
}

func writeManagedPlanFile(t *testing.T, plan managedartifact.Plan) string {
	t.Helper()
	data, err := plan.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "managed-plan.json")
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
