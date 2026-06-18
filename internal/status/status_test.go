package status

import (
	"strings"
	"testing"

	"repoctl/internal/config"
)

type fakeGitClient struct {
	ensureMirrorCalls []struct {
		path, url string
	}
	configureRemoteCalls []struct {
		repoPath, name, url string
	}
	fetchCalls []struct {
		repoPath, name string
	}
	setRemoteHeadCalls []struct {
		repoPath, name string
	}
	revListOutput string
	revParseShort  string
}

func (f *fakeGitClient) EnsureMirror(path, canonicalURL string) error {
	f.ensureMirrorCalls = append(f.ensureMirrorCalls, struct {
		path, url string
	}{path, canonicalURL})
	return nil
}

func (f *fakeGitClient) ConfigureRemote(repoPath, name, url string) error {
	f.configureRemoteCalls = append(f.configureRemoteCalls, struct {
		repoPath, name, url string
	}{repoPath, name, url})
	return nil
}

func (f *fakeGitClient) Fetch(repoPath, name string) error {
	f.fetchCalls = append(f.fetchCalls, struct {
		repoPath, name string
	}{repoPath, name})
	return nil
}

func (f *fakeGitClient) SetRemoteHead(repoPath, name string) error {
	f.setRemoteHeadCalls = append(f.setRemoteHeadCalls, struct {
		repoPath, name string
	}{repoPath, name})
	return nil
}

func (f *fakeGitClient) RevListLeftRightCount(repoPath, left, right string) (string, error) {
	return f.revListOutput, nil
}

func (f *fakeGitClient) RevParseShort(repoPath, rev string) (string, error) {
	return f.revParseShort, nil
}

func TestInterpretDivergence(t *testing.T) {
	tests := []struct {
		name       string
		left       int
		right      int
		wantState  State
		wantAhead  int
		wantBehind int
	}{
		{name: "equal", left: 0, right: 0, wantState: StateEqual, wantAhead: 0, wantBehind: 0},
		{name: "mirror behind canonical", left: 3, right: 0, wantState: StateBehind, wantAhead: 0, wantBehind: 3},
		{name: "mirror ahead canonical", left: 0, right: 2, wantState: StateAhead, wantAhead: 2, wantBehind: 0},
		{name: "diverged", left: 4, right: 5, wantState: StateDiverged, wantAhead: 5, wantBehind: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InterpretDivergence(tt.left, tt.right)
			if got.State != tt.wantState || got.Ahead != tt.wantAhead || got.Behind != tt.wantBehind {
				t.Fatalf("InterpretDivergence(%d, %d) = %#v", tt.left, tt.right, got)
			}
		})
	}
}

func TestParseRevListCount(t *testing.T) {
	left, right, err := ParseRevListCount("3\t0\n")
	if err != nil {
		t.Fatalf("ParseRevListCount returned error: %v", err)
	}
	if left != 3 || right != 0 {
		t.Fatalf("ParseRevListCount = %d, %d; want 3, 0", left, right)
	}
}

func TestCheckReturnsEqualState(t *testing.T) {
	git := &fakeGitClient{
		revListOutput: "0\t0\n",
		revParseShort:  "abc123",
	}

	repo := config.Repo{
		ID: "payments-api",
		UID: "repo.org.payments-api",
		Canonical: config.Endpoint{Provider: "gitlab", URL: "git@gitlab.com:org/payments-api.git"},
		Mirrors: []config.Endpoint{{Provider: "github", URL: "git@github.com:org/payments-api.git"}},
		Mode: "mirror",
	}

	result, err := Check(repo, git)
	if err != nil {
		t.Fatalf("Check returned unexpected error: %v", err)
	}
	if result.ID != "payments-api" || result.UID != "repo.org.payments-api" {
		t.Fatalf("identity = %q/%q, want payments-api/repo.org.payments-api", result.ID, result.UID)
	}
	if len(git.ensureMirrorCalls) != 1 || !strings.Contains(git.ensureMirrorCalls[0].path, "repo.org.payments-api.git") {
		t.Fatalf("ensure mirror path = %#v, want durable uid path", git.ensureMirrorCalls)
	}
	if result.State != StateEqual {
		t.Fatalf("state = %q, want %q", result.State, StateEqual)
	}
	if result.Ahead != 0 || result.Behind != 0 {
		t.Fatalf("counts = ahead %d behind %d, want 0 0", result.Ahead, result.Behind)
	}
	if result.Canonical != "abc123" || result.Mirror != "abc123" {
		t.Fatalf("canonical/mirror = %q/%q, want abc123/abc123", result.Canonical, result.Mirror)
	}
}

func TestCheckReturnsBehindState(t *testing.T) {
	git := &fakeGitClient{
		revListOutput: "4\t0\n",
		revParseShort:  "abc123",
	}

	repo := config.Repo{
		ID: "payments-api",
		UID: "repo.org.payments-api",
		Canonical: config.Endpoint{Provider: "gitlab", URL: "git@gitlab.com:org/payments-api.git"},
		Mirrors: []config.Endpoint{{Provider: "github", URL: "git@github.com:org/payments-api.git"}},
		Mode: "mirror",
	}

	result, err := Check(repo, git)
	if err != nil {
		t.Fatalf("Check returned unexpected error: %v", err)
	}
	if result.State != StateBehind {
		t.Fatalf("state = %q, want %q", result.State, StateBehind)
	}
	if result.Behind != 4 {
		t.Fatalf("behind = %d, want 4", result.Behind)
	}
}
