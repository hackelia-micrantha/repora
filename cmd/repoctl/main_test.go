package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"repoctl/internal/apply"
	"repoctl/internal/config"
	"repoctl/internal/status"
)

func TestStatusPrintsResultsInConfigOrderWhenChecksCompleteOutOfOrder(t *testing.T) {
	configPath := writeConfig(t, `repos:
  - id: slow-repo
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/slow-repo.git
    mirrors:
      - provider: github
        url: git@github.com:org/slow-repo.git
  - id: fast-repo
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/fast-repo.git
    mirrors:
      - provider: github
        url: git@github.com:org/fast-repo.git
`)

	oldCheck := statusCheck
	statusCheck = func(repo config.Repo) (status.Result, error) {
		if repo.ID == "slow-repo" {
			time.Sleep(25 * time.Millisecond)
		}
		return status.Result{ID: repo.ID, State: status.StateEqual}, nil
	}
	t.Cleanup(func() { statusCheck = oldCheck })

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"status", "-f", configPath, "--parallel", "2"})
	})

	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	slowIndex := strings.Index(stdout.String(), "slow-repo")
	fastIndex := strings.Index(stdout.String(), "fast-repo")
	if slowIndex == -1 || fastIndex == -1 {
		t.Fatalf("stdout missing repo ids:\n%s", stdout.String())
	}
	if slowIndex > fastIndex {
		t.Fatalf("repos printed out of config order:\n%s", stdout.String())
	}
}

func TestStatusStopsPrintingWhenRepoFailsWithoutContinueOnError(t *testing.T) {
	configPath := writeConfig(t, `repos:
  - id: broken-repo
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/broken-repo.git
    mirrors:
      - provider: github
        url: git@github.com:org/broken-repo.git
`)

	oldCheck := statusCheck
	statusCheck = func(repo config.Repo) (status.Result, error) {
		return status.Result{}, errors.New("fetch failed")
	}
	t.Cleanup(func() { statusCheck = oldCheck })

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"status", "-f", configPath})
	})

	if code != 1 {
		t.Fatalf("run returned %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty output on failure", stdout.String())
	}
}

func TestStatusJSONMatchesDocumentedShape(t *testing.T) {
	configPath := writeConfig(t, `repos:
  - id: payments-api
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/payments-api.git
    mirrors:
      - provider: github
        url: git@github.com:org/payments-api.git
`)

	oldCheck := statusCheck
	statusCheck = func(repo config.Repo) (status.Result, error) {
		return status.Result{
			ID:        repo.ID,
			State:     status.StateBehind,
			Ahead:     0,
			Behind:    3,
			Canonical: "abc1234",
			Mirror:    "def5678",
		}, nil
	}
	t.Cleanup(func() { statusCheck = oldCheck })

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"status", "-f", configPath, "--json"})
	})

	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}

	var got struct {
		Repos []struct {
			ID        string `json:"id"`
			Canonical struct {
				Ref    string `json:"ref"`
				Commit string `json:"commit"`
			} `json:"canonical"`
			Mirrors []struct {
				Provider string       `json:"provider"`
				Ref      string       `json:"ref"`
				Commit   string       `json:"commit"`
				State    status.State `json:"state"`
				Ahead    int          `json:"ahead"`
				Behind   int          `json:"behind"`
			} `json:"mirrors"`
		} `json:"repos"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal json: %v\n%s", err, stdout.String())
	}

	if len(got.Repos) != 1 {
		t.Fatalf("repo count = %d, want 1", len(got.Repos))
	}
	repo := got.Repos[0]
	if repo.ID != "payments-api" || repo.Canonical.Ref != "HEAD" || repo.Canonical.Commit != "abc1234" {
		t.Fatalf("canonical repo output = %#v", repo)
	}
	if len(repo.Mirrors) != 1 {
		t.Fatalf("mirror count = %d, want 1", len(repo.Mirrors))
	}
	mirror := repo.Mirrors[0]
	if mirror.Provider != "github" || mirror.Ref != "HEAD" || mirror.Commit != "def5678" {
		t.Fatalf("mirror identity output = %#v", mirror)
	}
	if mirror.State != status.StateBehind || mirror.Ahead != 0 || mirror.Behind != 3 {
		t.Fatalf("mirror status output = %#v", mirror)
	}
}

func TestPlanJSONIncludesPushMirrorActionForBehindMirror(t *testing.T) {
	configPath := writeConfig(t, `repos:
  - id: payments-api
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/payments-api.git
    mirrors:
      - provider: github
        url: git@github.com:org/payments-api.git
`)

	oldCheck := statusCheck
	statusCheck = func(repo config.Repo) (status.Result, error) {
		return status.Result{ID: repo.ID, State: status.StateBehind, Behind: 3}, nil
	}
	t.Cleanup(func() { statusCheck = oldCheck })

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"plan", "-f", configPath, "--json"})
	})

	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}

	var got struct {
		Plan []struct {
			ID      string `json:"id"`
			Actions []struct {
				Type        string `json:"type"`
				Target      string `json:"target"`
				Behind      int    `json:"behind"`
				Destructive bool   `json:"destructive"`
			} `json:"actions"`
		} `json:"plan"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal json: %v\n%s", err, stdout.String())
	}
	if len(got.Plan) != 1 {
		t.Fatalf("plan count = %d, want 1", len(got.Plan))
	}
	repoPlan := got.Plan[0]
	if repoPlan.ID != "payments-api" {
		t.Fatalf("plan id = %q, want payments-api", repoPlan.ID)
	}
	if len(repoPlan.Actions) != 1 {
		t.Fatalf("action count = %d, want 1", len(repoPlan.Actions))
	}
	action := repoPlan.Actions[0]
	if action.Type != "PUSH_MIRROR" || action.Target != "github" || action.Behind != 3 || action.Destructive {
		t.Fatalf("action = %#v, want non-destructive PUSH_MIRROR to github behind 3", action)
	}
}

func TestPlanHumanOutputDescribesMirrorPush(t *testing.T) {
	configPath := writeConfig(t, `repos:
  - id: payments-api
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/payments-api.git
    mirrors:
      - provider: github
        url: git@github.com:org/payments-api.git
`)

	oldCheck := statusCheck
	statusCheck = func(repo config.Repo) (status.Result, error) {
		return status.Result{ID: repo.ID, State: status.StateBehind, Behind: 3}, nil
	}
	t.Cleanup(func() { statusCheck = oldCheck })

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"plan", "-f", configPath})
	})

	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	want := "payments-api\n  push mirror github: 3 commits\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestPlanHumanOutputShowsNoChangesForEqualMirror(t *testing.T) {
	configPath := writeConfig(t, `repos:
  - id: payments-api
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/payments-api.git
    mirrors:
      - provider: github
        url: git@github.com:org/payments-api.git
`)

	oldCheck := statusCheck
	statusCheck = func(repo config.Repo) (status.Result, error) {
		return status.Result{ID: repo.ID, State: status.StateEqual}, nil
	}
	t.Cleanup(func() { statusCheck = oldCheck })

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"plan", "-f", configPath})
	})

	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	want := "payments-api\n  no changes\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestApplyHumanOutputPushesBehindMirror(t *testing.T) {
	configPath := writeConfig(t, `repos:
  - id: payments-api
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/payments-api.git
    mirrors:
      - provider: github
        url: git@github.com:org/payments-api.git
`)

	oldCheck := statusCheck
	statusCheck = func(repo config.Repo) (status.Result, error) {
		return status.Result{ID: repo.ID, State: status.StateBehind, Behind: 3}, nil
	}
	t.Cleanup(func() { statusCheck = oldCheck })

	oldApply := applyRepo
	applyRepo = func(repo config.Repo, result status.Result, force bool) (apply.RepoApply, error) {
		return apply.RepoApply{
			ID: repo.ID,
			Actions: []apply.Action{
				{Type: "PUSH_MIRROR", Target: repo.Mirrors[0].Provider},
			},
		}, nil
	}
	t.Cleanup(func() { applyRepo = oldApply })

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"apply", "-f", configPath})
	})

	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	want := "payments-api\n  push mirror github\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestApplyRefusesUnsafeMirrorBeforeSideEffects(t *testing.T) {
	configPath := writeConfig(t, `repos:
  - id: payments-api
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/payments-api.git
    mirrors:
      - provider: github
        url: git@github.com:org/payments-api.git
`)

	oldCheck := statusCheck
	statusCheck = func(repo config.Repo) (status.Result, error) {
		return status.Result{ID: repo.ID, State: status.StateDiverged, Ahead: 1, Behind: 2}, nil
	}
	t.Cleanup(func() { statusCheck = oldCheck })

	oldApply := applyRepo
	applyCalled := false
	applyRepo = func(repo config.Repo, result status.Result, force bool) (apply.RepoApply, error) {
		applyCalled = true
		return apply.RepoApply{}, nil
	}
	t.Cleanup(func() { applyRepo = oldApply })

	var stderr bytes.Buffer
	code := withStderr(t, &stderr, func() int {
		return run([]string{"apply", "-f", configPath})
	})

	if code != 2 {
		t.Fatalf("run returned %d, want 2", code)
	}
	if applyCalled {
		t.Fatal("applyRepo was called, want unsafe state rejected before side effects")
	}
	if !strings.Contains(stderr.String(), "--force") {
		t.Fatalf("stderr = %q, want --force guidance", stderr.String())
	}
}

func TestApplyPrintsResultsInConfigOrderWhenAppliesCompleteOutOfOrder(t *testing.T) {
	configPath := writeConfig(t, `repos:
  - id: slow-repo
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/slow-repo.git
    mirrors:
      - provider: github
        url: git@github.com:org/slow-repo.git
  - id: fast-repo
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/fast-repo.git
    mirrors:
      - provider: github
        url: git@github.com:org/fast-repo.git
`)

	oldCheck := statusCheck
	statusCheck = func(repo config.Repo) (status.Result, error) {
		return status.Result{ID: repo.ID, State: status.StateBehind, Behind: 1}, nil
	}
	t.Cleanup(func() { statusCheck = oldCheck })

	oldApply := applyRepo
	applyRepo = func(repo config.Repo, result status.Result, force bool) (apply.RepoApply, error) {
		if repo.ID == "slow-repo" {
			time.Sleep(25 * time.Millisecond)
		}
		return apply.RepoApply{
			ID:      repo.ID,
			Actions: []apply.Action{{Type: "PUSH_MIRROR", Target: repo.Mirrors[0].Provider}},
		}, nil
	}
	t.Cleanup(func() { applyRepo = oldApply })

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"apply", "-f", configPath, "--parallel", "2"})
	})

	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	slowIndex := strings.Index(stdout.String(), "slow-repo")
	fastIndex := strings.Index(stdout.String(), "fast-repo")
	if slowIndex == -1 || fastIndex == -1 {
		t.Fatalf("stdout missing repo ids:\n%s", stdout.String())
	}
	if slowIndex > fastIndex {
		t.Fatalf("repos printed out of config order:\n%s", stdout.String())
	}
}

func TestApplyHonorsParallelLimit(t *testing.T) {
	configPath := writeConfig(t, `repos:
  - id: one
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/one.git
    mirrors:
      - provider: github
        url: git@github.com:org/one.git
  - id: two
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/two.git
    mirrors:
      - provider: github
        url: git@github.com:org/two.git
`)

	oldCheck := statusCheck
	statusCheck = func(repo config.Repo) (status.Result, error) {
		return status.Result{ID: repo.ID, State: status.StateBehind, Behind: 1}, nil
	}
	t.Cleanup(func() { statusCheck = oldCheck })

	var mu sync.Mutex
	active := 0
	maxActive := 0
	oldApply := applyRepo
	applyRepo = func(repo config.Repo, result status.Result, force bool) (apply.RepoApply, error) {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()

		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		active--
		mu.Unlock()

		return apply.RepoApply{
			ID:      repo.ID,
			Actions: []apply.Action{{Type: "PUSH_MIRROR", Target: repo.Mirrors[0].Provider}},
		}, nil
	}
	t.Cleanup(func() { applyRepo = oldApply })

	code := run([]string{"apply", "-f", configPath, "--parallel", "1"})

	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if maxActive > 1 {
		t.Fatalf("max concurrent applies = %d, want <= 1", maxActive)
	}
}

func writeConfig(t *testing.T, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "repora.yaml")
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func withStdout(t *testing.T, dst *bytes.Buffer, fn func() int) int {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(dst, r)
		close(done)
	}()

	code := fn()
	_ = w.Close()
	<-done
	_ = r.Close()
	return code
}

func withStderr(t *testing.T, dst *bytes.Buffer, fn func() int) int {
	t.Helper()
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(dst, r)
		close(done)
	}()

	code := fn()
	_ = w.Close()
	<-done
	_ = r.Close()
	return code
}
