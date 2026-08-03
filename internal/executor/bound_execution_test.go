package executor

import (
	"errors"
	"strings"
	"testing"

	"repoctl/internal/planartifact"
)

func TestExecuteWithBindingsContinuesAfterRuntimeFailure(t *testing.T) {
	first := testAction(false)
	first.Source.Path = "org/payments-api"
	first.Target.Path = "one/payments-api"
	first.Target.Name = "serialized-first"
	second := first
	second.Target.Provider = "gitlab"
	second.Target.Path = "two/payments-api"
	second.Target.Name = "serialized-second"
	third := first
	third.Target.Path = "three/payments-api"
	third.Target.Name = "serialized-third"
	artifact, err := planartifact.FromCurrentPlans(testPlan(first, second, third))
	if err != nil {
		t.Fatal(err)
	}
	git := &fakeGit{pushErrs: map[int]error{1: errors.New("second mirror unavailable")}}

	got, err := ExecuteWithBindings("/tmp/repo", artifact, git, RuntimeBindings{
		SourceRemote: "canonical",
		TargetRemotes: map[string]string{
			"github:one/payments-api":   "mirror-0",
			"gitlab:two/payments-api":   "mirror-1",
			"github:three/payments-api": "mirror-2",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "second mirror unavailable") {
		t.Fatalf("error = %v, want aggregate runtime failure", err)
	}
	if len(git.pushCalls) != 3 {
		t.Fatalf("push calls = %#v, want every mirror attempted", git.pushCalls)
	}
	if git.pushCalls[0].remote != "mirror-0" || git.pushCalls[1].remote != "mirror-1" || git.pushCalls[2].remote != "mirror-2" {
		t.Fatalf("push order = %#v, want deterministic runtime bindings", git.pushCalls)
	}
	if got.Actions[0].Outcome != OutcomeApplied || got.Actions[1].Outcome != OutcomeFailed || got.Actions[2].Outcome != OutcomeApplied {
		t.Fatalf("outcomes = %#v, want applied/failed/applied", got.Actions)
	}
	if got.AllApplied() {
		t.Fatal("AllApplied returned true for partial failure")
	}
}

func TestExecuteWithBindingsPreflightFailureAttemptsNoAction(t *testing.T) {
	first := testAction(false)
	first.Source.Path = "org/payments-api"
	first.Target.Path = "one/payments-api"
	second := first
	second.Target.Provider = "gitlab"
	second.Target.Path = "two/payments-api"
	artifact, err := planartifact.FromCurrentPlans(testPlan(first, second))
	if err != nil {
		t.Fatal(err)
	}
	git := &fakeGit{resolveValues: map[string]string{
		"refs/remotes/mirror-1/main": strings.Repeat("b", 40),
	}}

	got, err := ExecuteWithBindings("/tmp/repo", artifact, git, RuntimeBindings{
		SourceRemote: "canonical",
		TargetRemotes: map[string]string{
			"github:one/payments-api": "mirror-0",
			"gitlab:two/payments-api": "mirror-1",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "stale action 1") {
		t.Fatalf("error = %v, want stale preflight", err)
	}
	assertNoMutation(t, git)
	if got.Actions[0].Outcome != OutcomeSkipped || got.Actions[1].Outcome != OutcomeFailed {
		t.Fatalf("outcomes = %#v, want skipped/stale-failed", got.Actions)
	}
}
