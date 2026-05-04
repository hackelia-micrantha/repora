package apply

import (
	"strings"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/status"
)

type fakeGit struct {
	syncCalls []struct {
		repoPath string
		remote   string
	}
	pushCalls []struct {
		repoPath string
		remote   string
	}
}

func (f *fakeGit) SyncMirrorFromRemote(repoPath, remote string) error {
	f.syncCalls = append(f.syncCalls, struct {
		repoPath string
		remote   string
	}{repoPath: repoPath, remote: remote})
	return nil
}

func (f *fakeGit) PushMirror(repoPath, remote string) error {
	f.pushCalls = append(f.pushCalls, struct {
		repoPath string
		remote   string
	}{repoPath: repoPath, remote: remote})
	return nil
}

func TestExecutePushesBehindMirror(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	result := status.Result{ID: repo.ID, State: status.StateBehind, Behind: 3}

	got, err := Execute(repo, result, git, false)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(git.syncCalls) != 1 || git.syncCalls[0].remote != "canonical" {
		t.Fatalf("sync calls = %#v, want one canonical sync", git.syncCalls)
	}
	if len(git.pushCalls) != 1 || git.pushCalls[0].remote != "mirror" {
		t.Fatalf("push calls = %#v, want one mirror push", git.pushCalls)
	}
	if len(got.Actions) != 1 {
		t.Fatalf("action count = %d, want 1", len(got.Actions))
	}
	action := got.Actions[0]
	if action.Type != "PUSH_MIRROR" || action.Target != "github" || action.Forced || action.Destructive {
		t.Fatalf("action = %#v, want safe PUSH_MIRROR to github", action)
	}
}

func TestExecuteRefusesUnsafeMirrorWithoutForce(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	result := status.Result{ID: repo.ID, State: status.StateDiverged, Ahead: 1, Behind: 2}

	_, err := Execute(repo, result, git, false)
	if err == nil {
		t.Fatal("Execute returned nil error, want unsafe state rejection")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("error = %q, want --force guidance", err.Error())
	}
	if len(git.syncCalls) != 0 || len(git.pushCalls) != 0 {
		t.Fatalf("git calls = sync %#v push %#v, want no side effects", git.syncCalls, git.pushCalls)
	}
}

func TestExecuteForcePushesUnsafeMirror(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	result := status.Result{ID: repo.ID, State: status.StateAhead, Ahead: 1}

	got, err := Execute(repo, result, git, true)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(git.syncCalls) != 1 || len(git.pushCalls) != 1 {
		t.Fatalf("git calls = sync %#v push %#v, want one sync and push", git.syncCalls, git.pushCalls)
	}
	if len(got.Actions) != 1 || !got.Actions[0].Forced || !got.Actions[0].Destructive {
		t.Fatalf("actions = %#v, want forced destructive push", got.Actions)
	}
}

func TestExecuteNoopsEqualMirror(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	result := status.Result{ID: repo.ID, State: status.StateEqual}

	got, err := Execute(repo, result, git, false)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(git.syncCalls) != 0 || len(git.pushCalls) != 0 {
		t.Fatalf("git calls = sync %#v push %#v, want no side effects", git.syncCalls, git.pushCalls)
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
