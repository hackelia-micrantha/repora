package apply

import (
	"strings"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/status"
)

type fakeGit struct {
	resolveRemoteHeadBranchCalls []string
	resolveRevisionCalls         []string
	pushBranchCalls              []struct {
		remote    string
		srcRef    string
		dstBranch string
	}
	forcePushBranchWithLeaseCalls []struct {
		remote         string
		srcRef         string
		dstBranch      string
		expectedOldOID string
	}
}

func (f *fakeGit) ResolveRemoteHeadBranch(repoPath, remote string) (string, error) {
	f.resolveRemoteHeadBranchCalls = append(f.resolveRemoteHeadBranchCalls, remote)
	return "main", nil
}

func (f *fakeGit) ResolveRevision(repoPath, rev string) (string, error) {
	f.resolveRevisionCalls = append(f.resolveRevisionCalls, rev)
	return "abc123456789", nil
}

func (f *fakeGit) PushBranch(repoPath, remote, srcRef, dstBranch string) error {
	f.pushBranchCalls = append(f.pushBranchCalls, struct {
		remote    string
		srcRef    string
		dstBranch string
	}{remote: remote, srcRef: srcRef, dstBranch: dstBranch})
	return nil
}

func (f *fakeGit) ForcePushBranchWithLease(repoPath, remote, srcRef, dstBranch, expectedOldOID string) error {
	f.forcePushBranchWithLeaseCalls = append(f.forcePushBranchWithLeaseCalls, struct {
		remote         string
		srcRef         string
		dstBranch      string
		expectedOldOID string
	}{remote: remote, srcRef: srcRef, dstBranch: dstBranch, expectedOldOID: expectedOldOID})
	return nil
}

func TestExecutePushesBehindMirror(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	st := status.Result{ID: repo.ID, State: status.StateBehind, Behind: 3}

	got, err := Execute(repo, st, git, false, false)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(git.pushBranchCalls) != 1 || git.pushBranchCalls[0].remote != "mirror" {
		t.Fatalf("push calls = %#v, want one mirror push", git.pushBranchCalls)
	}
	if len(got.Actions) != 1 {
		t.Fatalf("action count = %d, want 1", len(got.Actions))
	}
	action := got.Actions[0]
	if action.Type != "PUSH_BRANCH" || action.Target != "github/main" || action.Force {
		t.Fatalf("action = %#v, want safe PUSH_BRANCH to github/main", action)
	}
}

func TestExecuteRefusesUnsafeMirrorWithoutForce(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	st := status.Result{ID: repo.ID, State: status.StateDiverged, Ahead: 1, Behind: 2}

	_, err := Execute(repo, st, git, false, false)
	if err == nil {
		t.Fatal("Execute returned nil error, want unsafe state rejection")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("error = %q, want --force guidance", err.Error())
	}
	if len(git.pushBranchCalls) != 0 && len(git.forcePushBranchWithLeaseCalls) != 0 {
		t.Fatalf("git calls = push %#v force %#v, want no side effects", git.pushBranchCalls, git.forcePushBranchWithLeaseCalls)
	}
}

func TestExecuteForcePushesUnsafeMirror(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	st := status.Result{ID: repo.ID, State: status.StateAhead, Ahead: 1}

	got, err := Execute(repo, st, git, true, false)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(git.forcePushBranchWithLeaseCalls) != 1 {
		t.Fatalf("git calls = force %#v, want one force push", git.forcePushBranchWithLeaseCalls)
	}
	if len(got.Actions) != 1 || !got.Actions[0].Force {
		t.Fatalf("actions = %#v, want forced push", got.Actions)
	}
}

func TestExecuteNoopsEqualMirror(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	st := status.Result{ID: repo.ID, State: status.StateEqual}

	got, err := Execute(repo, st, git, false, false)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(git.pushBranchCalls) != 0 || len(git.forcePushBranchWithLeaseCalls) != 0 {
		t.Fatalf("git calls = push %#v force %#v, want no side effects", git.pushBranchCalls, git.forcePushBranchWithLeaseCalls)
	}
	if len(got.Actions) != 0 {
		t.Fatalf("actions = %#v, want none", got.Actions)
	}
}

func testRepo() config.Repo {
	return config.Repo{
		ID:        "payments-api",
		Canonical: config.Endpoint{Provider: "gitlab", URL: "git@gitlab.com:org/payments-api.git"},
		Mirrors: []config.Endpoint{
			{Provider: "github", URL: "git@github.com:org/payments-api.git"},
		},
		Mode: "mirror",
	}
}
