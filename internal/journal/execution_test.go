package journal

import (
	"errors"
	"strings"
	"testing"

	"repoctl/internal/executor"
	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
)

func TestFromPreflightProjectsValidatedResult(t *testing.T) {
	artifact := testArtifact()
	planned := testPlan().Actions[0]
	record, err := FromPreflight("run-dry", artifact, executor.Result{Actions: []executor.ActionResult{{
		Index:   0,
		Action:  planned,
		Outcome: executor.OutcomeSkipped,
	}}}, nil)
	if err != nil {
		t.Fatalf("FromPreflight returned error: %v", err)
	}
	if record.Phase != PhaseResult || record.Mode != ModeDryRun || record.Actions[0].Outcome != OutcomeValidated {
		t.Fatalf("record = %#v, want validated dry-run result", record)
	}
}

func TestFromPreflightProjectsStaleAndSkippedResults(t *testing.T) {
	planned := twoActionPlan()
	artifact := planartifact.FromPlans(planned)
	record, err := FromPreflight("run-stale", artifact, executor.Result{Actions: []executor.ActionResult{
		{Index: 0, Action: planned.Actions[0], Outcome: executor.OutcomeFailed, Stale: true, Error: "stale action 0: target changed"},
		{Index: 1, Action: planned.Actions[1], Outcome: executor.OutcomeSkipped},
	}}, errors.New("stale action"))
	if err != nil {
		t.Fatalf("FromPreflight returned error: %v", err)
	}
	if record.Actions[0].Outcome != OutcomeStale || record.Actions[1].Outcome != OutcomeSkipped {
		t.Fatalf("outcomes = %q, %q, want STALE, SKIPPED", record.Actions[0].Outcome, record.Actions[1].Outcome)
	}
}

func TestFromExecutionProjectsAppliedResult(t *testing.T) {
	artifact := testArtifact()
	planned := testPlan().Actions[0]
	record, err := FromExecution("run-apply", artifact, executor.Result{Actions: []executor.ActionResult{{
		Index:    0,
		Action:   planned,
		Outcome:  executor.OutcomeApplied,
		AfterOID: testSourceOID,
	}}})
	if err != nil {
		t.Fatalf("FromExecution returned error: %v", err)
	}
	if record.Phase != PhaseResult || record.Mode != ModeApply || record.Actions[0].Outcome != OutcomeApplied || record.Actions[0].After != testSourceOID {
		t.Fatalf("record = %#v, want applied result evidence with after OID", record)
	}
}

func TestFromExecutionProjectsStaleAndSkippedResults(t *testing.T) {
	planned := twoActionPlan()
	artifact := planartifact.FromPlans(planned)
	record, err := FromExecution("run-stale", artifact, executor.Result{Actions: []executor.ActionResult{
		{Index: 0, Action: planned.Actions[0], Outcome: executor.OutcomeFailed, Stale: true, Error: "stale action 0: target changed"},
		{Index: 1, Action: planned.Actions[1], Outcome: executor.OutcomeSkipped},
	}})
	if err != nil {
		t.Fatalf("FromExecution returned error: %v", err)
	}
	if record.Actions[0].Outcome != OutcomeStale || record.Actions[1].Outcome != OutcomeSkipped {
		t.Fatalf("outcomes = %q, %q, want STALE, SKIPPED", record.Actions[0].Outcome, record.Actions[1].Outcome)
	}
}

func TestFromExecutionPreservesPartialFailure(t *testing.T) {
	planned := twoActionPlan()
	artifact := planartifact.FromPlans(planned)
	record, err := FromExecution("run-partial", artifact, executor.Result{Actions: []executor.ActionResult{
		{Index: 0, Action: planned.Actions[0], Outcome: executor.OutcomeApplied, AfterOID: testSourceOID},
		{Index: 1, Action: planned.Actions[1], Outcome: executor.OutcomeFailed, Error: "remote rejected update"},
	}})
	if err != nil {
		t.Fatalf("FromExecution returned error: %v", err)
	}
	if record.Actions[0].Outcome != OutcomeApplied || record.Actions[0].After != testSourceOID {
		t.Fatalf("first action = %#v, want applied evidence", record.Actions[0])
	}
	if record.Actions[1].Outcome != OutcomeFailed || record.Actions[1].Error != "remote rejected update" {
		t.Fatalf("second action = %#v, want failed evidence", record.Actions[1])
	}
}

func TestFromExecutionRedactsUnsafeDiagnostic(t *testing.T) {
	planned := testPlan().Actions[0]
	record, err := FromExecution("run-failed", testArtifact(), executor.Result{Actions: []executor.ActionResult{{
		Index:   0,
		Action:  planned,
		Outcome: executor.OutcomeFailed,
		Error:   "push failed token=secret",
	}}})
	if err != nil {
		t.Fatalf("FromExecution returned error: %v", err)
	}
	if record.Actions[0].Error != "execution diagnostic redacted" {
		t.Fatalf("error = %q, want redacted diagnostic", record.Actions[0].Error)
	}
}

func TestResultProjectionRejectsIncompleteOrMismatchedEvidence(t *testing.T) {
	planned := testPlan().Actions[0]
	tests := []struct {
		name   string
		result executor.Result
		want   string
	}{
		{name: "missing action", result: executor.Result{}, want: "has 0 actions"},
		{name: "wrong index", result: executor.Result{Actions: []executor.ActionResult{{Index: 1, Action: planned, Outcome: executor.OutcomeSkipped}}}, want: "index"},
		{name: "wrong action", result: executor.Result{Actions: []executor.ActionResult{{Index: 0, Action: plan.PlannedAction{}, Outcome: executor.OutcomeSkipped}}}, want: "does not match"},
		{name: "applied without after", result: executor.Result{Actions: []executor.ActionResult{{Index: 0, Action: planned, Outcome: executor.OutcomeApplied}}}, want: "requires after"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromExecution("run-invalid", testArtifact(), tt.result)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func twoActionPlan() plan.ReconciliationPlan {
	planned := testPlan()
	second := planned.Actions[0]
	second.Target = plan.Remote{Provider: "bitbucket", Name: "backup", Branch: "main"}
	second.ExpectedOldTarget = "3333333333333333333333333333333333333333"
	planned.Actions = append(planned.Actions, second)
	return planned
}
