package executor

import (
	"context"
	"strings"
	"testing"

	"repoctl/internal/plan"
)

func TestInterruptedExecutionDoesNotReportCompletionAndCanRetry(t *testing.T) {
	artifact := testArtifact(testAction(false))
	interrupted := &fakeGit{pushErrs: map[int]error{0: context.Canceled}}

	first, err := Execute("/tmp/repo", artifact, interrupted)
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("error = %v, want interruption", err)
	}
	if len(first.Actions) != 1 || first.Actions[0].Outcome != OutcomeFailed || first.AllApplied() {
		t.Fatalf("first result = %#v, want failed non-complete operation", first)
	}

	retry := &fakeGit{}
	second, err := Execute("/tmp/repo", artifact, retry)
	if err != nil {
		t.Fatalf("retry returned error: %v", err)
	}
	if len(second.Actions) != 1 || second.Actions[0].Outcome != OutcomeApplied || !second.AllApplied() {
		t.Fatalf("retry result = %#v, want successful retry", second)
	}
}

func TestFailureBetweenMutationStepsPreservesRecoveryBoundary(t *testing.T) {
	first := testAction(false)
	first.Target.Branch = "main"
	second := testAction(false)
	second.Target.Branch = "release"
	third := testAction(false)
	third.Target.Branch = "next"

	git := &fakeGit{pushErrs: map[int]error{1: context.DeadlineExceeded}}
	got, err := Execute("/tmp/repo", testArtifact(first, second, third), git)
	if err == nil || !strings.Contains(err.Error(), "execute action 1") {
		t.Fatalf("error = %v, want second-step failure", err)
	}
	want := []Outcome{OutcomeApplied, OutcomeFailed, OutcomeSkipped}
	for i, outcome := range want {
		if got.Actions[i].Outcome != outcome {
			t.Fatalf("action %d = %#v, want %s", i, got.Actions[i], outcome)
		}
	}
	if got.AllApplied() || len(git.pushCalls) != 2 {
		t.Fatalf("result = %#v calls = %#v, want explicit incomplete boundary", got, git.pushCalls)
	}

	remaining := []plan.PlannedAction{second, third}
	retry := &fakeGit{}
	retried, err := Execute("/tmp/repo", testArtifact(remaining...), retry)
	if err != nil {
		t.Fatalf("recovery retry returned error: %v", err)
	}
	if !retried.AllApplied() || len(retry.pushCalls) != 2 {
		t.Fatalf("retry result = %#v calls = %#v, want remaining actions applied", retried, retry.pushCalls)
	}
}
