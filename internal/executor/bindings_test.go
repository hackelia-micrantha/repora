package executor

import (
	"strings"
	"testing"

	"repoctl/internal/planartifact"
)

func TestPreflightWithBindingsUsesCurrentAliasesWithoutChangingPlan(t *testing.T) {
	action := testAction(false)
	action.Source.Path = "org/payments-api"
	action.Target.Path = "org/payments-api"
	action.Target.Name = "mirror-0"
	artifact, err := planartifact.FromCurrentPlans(testPlan(action))
	if err != nil {
		t.Fatal(err)
	}
	git := &fakeGit{}

	got, err := PreflightWithBindings("/tmp/repo", artifact, git, RuntimeBindings{
		SourceRemote: "canonical",
		TargetRemotes: map[string]string{
			"github:org/payments-api": "mirror-1",
		},
	})
	if err != nil {
		t.Fatalf("PreflightWithBindings returned error: %v", err)
	}
	if strings.Join(git.resolveCalls, ",") != "refs/remotes/canonical/main,refs/remotes/mirror-1/main" {
		t.Fatalf("resolve calls = %#v, want current runtime aliases", git.resolveCalls)
	}
	if len(got.Actions) != 1 || got.Actions[0].Action.Target.Name != "mirror-0" || got.Actions[0].Outcome != OutcomeSkipped {
		t.Fatalf("result = %#v, want unchanged reviewed action", got)
	}
	assertNoMutation(t, git)
}

func TestPreflightWithBindingsRejectsUnknownTargetBeforeReads(t *testing.T) {
	action := testAction(false)
	action.Source.Path = "org/payments-api"
	action.Target.Path = "other/payments-api"
	artifact, err := planartifact.FromCurrentPlans(testPlan(action))
	if err != nil {
		t.Fatal(err)
	}
	git := &fakeGit{}

	got, err := PreflightWithBindings("/tmp/repo", artifact, git, RuntimeBindings{
		SourceRemote:  "canonical",
		TargetRemotes: map[string]string{"github:org/payments-api": "mirror-0"},
	})
	if err == nil || !strings.Contains(err.Error(), "no runtime binding") {
		t.Fatalf("error = %v, want missing binding rejection", err)
	}
	if len(git.resolveCalls) != 0 {
		t.Fatalf("resolve calls = %#v, want no Git reads", git.resolveCalls)
	}
	if len(got.Actions) != 1 || got.Actions[0].Outcome != OutcomeFailed || got.Actions[0].Stale {
		t.Fatalf("result = %#v, want structural binding failure", got)
	}
}

func TestPreflightWithBindingsChecksAllTargetsBeforeMutation(t *testing.T) {
	first := testAction(false)
	first.Source.Path = "org/payments-api"
	first.Target.Path = "org/payments-api"
	first.Target.Name = "mirror-0"
	second := first
	second.Target.Provider = "gitlab"
	second.Target.Path = "backup/payments-api"
	second.Target.Name = "mirror-1"
	artifact, err := planartifact.FromCurrentPlans(testPlan(first, second))
	if err != nil {
		t.Fatal(err)
	}
	git := &fakeGit{resolveValues: map[string]string{
		"refs/remotes/mirror-1/main": strings.Repeat("b", 40),
	}}

	got, err := PreflightWithBindings("/tmp/repo", artifact, git, RuntimeBindings{
		SourceRemote: "canonical",
		TargetRemotes: map[string]string{
			"github:org/payments-api":    "mirror-0",
			"gitlab:backup/payments-api": "mirror-1",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "stale action 1") {
		t.Fatalf("error = %v, want second target stale failure", err)
	}
	if got.Actions[0].Outcome != OutcomeSkipped || got.Actions[1].Outcome != OutcomeFailed {
		t.Fatalf("result = %#v, want skipped then failed", got)
	}
	assertNoMutation(t, git)
}
