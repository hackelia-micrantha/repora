package executor

import (
	"errors"
	"strings"
	"testing"

	"repoctl/internal/plan"
)

type fakeGit struct {
	pushCalls []struct {
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
	pushErr  error
	forceErr error
}

func (f *fakeGit) PushBranch(_ string, remote, srcRef, dstBranch string) error {
	f.pushCalls = append(f.pushCalls, struct {
		remote    string
		srcRef    string
		dstBranch string
	}{remote: remote, srcRef: srcRef, dstBranch: dstBranch})
	return f.pushErr
}

func (f *fakeGit) ForcePushBranchWithLease(_ string, remote, srcRef, dstBranch, expectedOldOID string) error {
	f.forceCalls = append(f.forceCalls, struct {
		remote         string
		srcRef         string
		dstBranch      string
		expectedOldOID string
	}{remote: remote, srcRef: srcRef, dstBranch: dstBranch, expectedOldOID: expectedOldOID})
	return f.forceErr
}

func TestExecuteNormalPushUsesPlannedArguments(t *testing.T) {
	git := &fakeGit{}
	planned := testPlan(testAction(false))

	got, err := Execute("/tmp/repo", planned, git)
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
	if len(got.Actions) != 1 || !got.Actions[0].Applied {
		t.Fatalf("result = %#v, want one applied action", got)
	}
}

func TestExecuteForcedPushUsesPlannedLease(t *testing.T) {
	git := &fakeGit{}
	planned := testPlan(testAction(true))

	_, err := Execute("/tmp/repo", planned, git)
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
}

func TestExecuteRejectsUnknownActionWithoutMutation(t *testing.T) {
	git := &fakeGit{}
	action := testAction(false)
	action.Type = plan.ActionType("UNKNOWN")

	_, err := Execute("/tmp/repo", testPlan(action), git)
	if err == nil || !strings.Contains(err.Error(), "unsupported action type") {
		t.Fatalf("error = %v, want unsupported action", err)
	}
	if len(git.pushCalls) != 0 || len(git.forceCalls) != 0 {
		t.Fatalf("git calls = push %#v force %#v, want none", git.pushCalls, git.forceCalls)
	}
}

func TestExecuteRejectsMalformedForceWithoutMutation(t *testing.T) {
	git := &fakeGit{}
	action := testAction(true)
	action.ExpectedOldTarget = ""

	_, err := Execute("/tmp/repo", testPlan(action), git)
	if err == nil || !strings.Contains(err.Error(), "expected old target") {
		t.Fatalf("error = %v, want lease validation", err)
	}
	if len(git.pushCalls) != 0 || len(git.forceCalls) != 0 {
		t.Fatalf("git calls = push %#v force %#v, want none", git.pushCalls, git.forceCalls)
	}
}

func TestExecuteReturnsMutationFailure(t *testing.T) {
	git := &fakeGit{pushErr: errors.New("network failed")}

	got, err := Execute("/tmp/repo", testPlan(testAction(false)), git)
	if err == nil || !strings.Contains(err.Error(), "network failed") {
		t.Fatalf("error = %v, want mutation failure", err)
	}
	if len(got.Actions) != 0 {
		t.Fatalf("result = %#v, want no successful actions", got)
	}
}

func testPlan(actions ...plan.PlannedAction) plan.ReconciliationPlan {
	return plan.ReconciliationPlan{ID: "payments-api", UID: "repo.org.payments-api", Actions: actions}
}

func testAction(force bool) plan.PlannedAction {
	action := plan.PlannedAction{
		Type:   plan.ActionPushBranch,
		Source: plan.Remote{Name: "canonical", Branch: "main"},
		Target: plan.Remote{Name: "mirror", Provider: "github", Branch: "main"},
		Force:  force,
	}
	if force {
		action.ExpectedOldTarget = "abc123"
	}
	return action
}
