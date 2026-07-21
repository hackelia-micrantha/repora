package executor

import (
	"errors"
	"strings"
	"testing"

	"repoctl/internal/plan"
)

type fakeGit struct {
	resolveValues map[string]string
	resolveErrs   map[string]error
	resolveCalls  []string
	pushCalls     []struct {
		remote    string
		srcRef    string
		dstBranch string
	}
	forceCalls []struct {
		remote         string
		srcRef         string
		dstBranch      string
		expectedOldOID string
	}
	pushErrs  map[int]error
	forceErrs map[int]error
}

func (f *fakeGit) ResolveRevision(_ string, rev string) (string, error) {
	f.resolveCalls = append(f.resolveCalls, rev)
	if err := f.resolveErrs[rev]; err != nil {
		return "", err
	}
	if value, ok := f.resolveValues[rev]; ok {
		return value, nil
	}
	if strings.HasPrefix(rev, "refs/remotes/canonical/") {
		return "src123", nil
	}
	return "abc123", nil
}

func (f *fakeGit) PushBranch(_ string, remote, srcRef, dstBranch string) error {
	f.pushCalls = append(f.pushCalls, struct {
		remote    string
		srcRef    string
		dstBranch string
	}{remote: remote, srcRef: srcRef, dstBranch: dstBranch})
	return f.pushErrs[len(f.pushCalls)-1]
}

func (f *fakeGit) ForcePushBranchWithLease(_ string, remote, srcRef, dstBranch, expectedOldOID string) error {
	f.forceCalls = append(f.forceCalls, struct {
		remote         string
		srcRef         string
		dstBranch      string
		expectedOldOID string
	}{remote: remote, srcRef: srcRef, dstBranch: dstBranch, expectedOldOID: expectedOldOID})
	return f.forceErrs[len(f.forceCalls)-1]
}

func TestExecuteNormalPushUsesPlannedArguments(t *testing.T) {
	git := &fakeGit{}
	got, err := Execute("/tmp/repo", testPlan(testAction(false)), git)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(git.pushCalls) != 1 {
		t.Fatalf("push calls = %#v, want one", git.pushCalls)
	}
	call := git.pushCalls[0]
	if call.remote != "mirror" || call.srcRef != "refs/remotes/canonical/main" || call.dstBranch != "main" {
		t.Fatalf("push call = %#v, want planned arguments", call)
	}
	if len(got.Actions) != 1 || got.Actions[0].Index != 0 || got.Actions[0].Outcome != OutcomeApplied || !got.AllApplied() {
		t.Fatalf("result = %#v, want one applied action", got)
	}
}

func TestExecuteForcedPushUsesPlannedLease(t *testing.T) {
	git := &fakeGit{}
	got, err := Execute("/tmp/repo", testPlan(testAction(true)), git)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(git.forceCalls) != 1 {
		t.Fatalf("force calls = %#v, want one", git.forceCalls)
	}
	call := git.forceCalls[0]
	if call.remote != "mirror" || call.srcRef != "refs/remotes/canonical/main" || call.dstBranch != "main" || call.expectedOldOID != "abc123" {
		t.Fatalf("force call = %#v, want planned arguments", call)
	}
	if !got.AllApplied() {
		t.Fatalf("result = %#v, want applied forced action", got)
	}
}

func TestExecuteRejectsUnknownActionWithoutMutation(t *testing.T) {
	git := &fakeGit{}
	action := testAction(false)
	action.Type = plan.ActionType("UNKNOWN")

	got, err := Execute("/tmp/repo", testPlan(action), git)
	if err == nil || !strings.Contains(err.Error(), "unsupported action type") {
		t.Fatalf("error = %v, want unsupported action", err)
	}
	if got.Actions[0].Outcome != OutcomeFailed || got.Actions[0].Error == "" {
		t.Fatalf("result = %#v, want failed validation result", got)
	}
	assertNoMutation(t, git)
}

func TestExecuteValidatesCompletePlanBeforeMutation(t *testing.T) {
	git := &fakeGit{}
	valid := testAction(false)
	invalid := testAction(true)
	invalid.ExpectedOldTarget = ""

	got, err := Execute("/tmp/repo", testPlan(valid, invalid), git)
	if err == nil || !strings.Contains(err.Error(), "validate action 1") {
		t.Fatalf("error = %v, want second-action validation failure", err)
	}
	if got.Actions[0].Outcome != OutcomeSkipped || got.Actions[1].Outcome != OutcomeFailed {
		t.Fatalf("result = %#v, want skipped then failed", got)
	}
	if len(git.resolveCalls) != 0 {
		t.Fatalf("resolve calls = %#v, want no stale checks before structural validation completes", git.resolveCalls)
	}
	assertNoMutation(t, git)
}

func TestExecuteRejectsStaleSourceWithoutMutation(t *testing.T) {
	git := &fakeGit{resolveValues: map[string]string{"refs/remotes/canonical/main": "new-source"}}

	got, err := Execute("/tmp/repo", testPlan(testAction(false)), git)
	if err == nil || !strings.Contains(err.Error(), "stale action 0") || !strings.Contains(err.Error(), "source") {
		t.Fatalf("error = %v, want stale source rejection", err)
	}
	if got.Actions[0].Outcome != OutcomeFailed {
		t.Fatalf("result = %#v, want failed stale action", got)
	}
	assertNoMutation(t, git)
}

func TestExecuteRejectsStaleTargetWithoutMutation(t *testing.T) {
	git := &fakeGit{resolveValues: map[string]string{"refs/remotes/mirror/main": "new-target"}}

	_, err := Execute("/tmp/repo", testPlan(testAction(true)), git)
	if err == nil || !strings.Contains(err.Error(), "target") || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("error = %v, want stale target rejection", err)
	}
	assertNoMutation(t, git)
}

func TestExecuteValidatesAllRefsBeforeMutation(t *testing.T) {
	first := testAction(false)
	second := testAction(false)
	second.Target.Branch = "release"
	git := &fakeGit{resolveValues: map[string]string{"refs/remotes/mirror/release": "changed"}}

	got, err := Execute("/tmp/repo", testPlan(first, second), git)
	if err == nil || !strings.Contains(err.Error(), "stale action 1") {
		t.Fatalf("error = %v, want second-action stale rejection", err)
	}
	if got.Actions[0].Outcome != OutcomeSkipped || got.Actions[1].Outcome != OutcomeFailed {
		t.Fatalf("result = %#v, want skipped then failed", got)
	}
	assertNoMutation(t, git)
}

func TestExecutePreservesPartialResultsAfterMutationFailure(t *testing.T) {
	git := &fakeGit{pushErrs: map[int]error{1: errors.New("network failed")}}
	actions := []plan.PlannedAction{testAction(false), testAction(false), testAction(false)}

	got, err := Execute("/tmp/repo", testPlan(actions...), git)
	if err == nil || !strings.Contains(err.Error(), "execute action 1") || !strings.Contains(err.Error(), "network failed") {
		t.Fatalf("error = %v, want second-action mutation failure", err)
	}
	if len(git.pushCalls) != 2 {
		t.Fatalf("push calls = %#v, want first and second only", git.pushCalls)
	}
	want := []Outcome{OutcomeApplied, OutcomeFailed, OutcomeSkipped}
	for i, outcome := range want {
		if got.Actions[i].Index != i || got.Actions[i].Outcome != outcome {
			t.Fatalf("action %d = %#v, want outcome %s", i, got.Actions[i], outcome)
		}
	}
	if got.Actions[1].Error != "network failed" || got.Actions[2].Error != "" || got.AllApplied() {
		t.Fatalf("result = %#v, want preserved failure and skipped tail", got)
	}
}

func TestExecuteSuccessfulMultiActionPlanPreservesOrder(t *testing.T) {
	git := &fakeGit{}
	first := testAction(false)
	first.Target.Branch = "main"
	second := testAction(false)
	second.Target.Branch = "release"

	got, err := Execute("/tmp/repo", testPlan(first, second), git)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(git.pushCalls) != 2 || git.pushCalls[0].dstBranch != "main" || git.pushCalls[1].dstBranch != "release" {
		t.Fatalf("push calls = %#v, want plan order", git.pushCalls)
	}
	if !got.AllApplied() {
		t.Fatalf("result = %#v, want all applied", got)
	}
	for i, action := range got.Actions {
		if action.Index != i || action.Outcome != OutcomeApplied {
			t.Fatalf("action %d = %#v, want ordered applied result", i, action)
		}
	}
}

func assertNoMutation(t *testing.T, git *fakeGit) {
	t.Helper()
	if len(git.pushCalls) != 0 || len(git.forceCalls) != 0 {
		t.Fatalf("git calls = push %#v force %#v, want none", git.pushCalls, git.forceCalls)
	}
}

func testPlan(actions ...plan.PlannedAction) plan.ReconciliationPlan {
	return plan.ReconciliationPlan{ID: "payments-api", UID: "repo.org.payments-api", Actions: actions}
}

func testAction(force bool) plan.PlannedAction {
	return plan.PlannedAction{
		Type:              plan.ActionPushBranch,
		Source:            plan.Remote{Name: "canonical", Branch: "main"},
		Target:            plan.Remote{Name: "mirror", Provider: "github", Branch: "main"},
		Force:             force,
		ExpectedSource:    "src123",
		ExpectedOldTarget: "abc123",
	}
}
