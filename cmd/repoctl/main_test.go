package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
