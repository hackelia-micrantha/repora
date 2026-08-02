package journal

import (
	"errors"
	"strings"
	"testing"

	"repoctl/internal/executor"
)

func TestFromPreflightRejectsInconsistentSuccessfulEvidence(t *testing.T) {
	planned := testPlan().Actions[0]
	_, err := FromPreflight("run-invalid", testArtifact(), executor.Result{Actions: []executor.ActionResult{{
		Index:   0,
		Action:  planned,
		Outcome: executor.OutcomeFailed,
		Error:   "unexpected failure",
	}}}, nil)
	if err == nil || !strings.Contains(err.Error(), "inconsistent executor evidence") {
		t.Fatalf("error = %v, want inconsistent preflight rejection", err)
	}
}

func TestFromPreflightRejectsFailureWithoutFailedAction(t *testing.T) {
	planned := testPlan().Actions[0]
	_, err := FromPreflight("run-invalid", testArtifact(), executor.Result{Actions: []executor.ActionResult{{
		Index:   0,
		Action:  planned,
		Outcome: executor.OutcomeSkipped,
	}}}, errors.New("preflight failed"))
	if err == nil || !strings.Contains(err.Error(), "does not identify a failed action") {
		t.Fatalf("error = %v, want missing failed action rejection", err)
	}
}
