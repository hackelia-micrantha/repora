package main

import (
	"bytes"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/status"
)

func TestPlanHumanOutputLabelsDestructiveOverwrite(t *testing.T) {
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
		return status.Result{ID: repo.ID, UID: repo.DurableID(), State: status.StateAhead, Ahead: 2}, nil
	}
	t.Cleanup(func() { statusCheck = oldCheck })

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"plan", "-f", configPath})
	})
	if code != 2 {
		t.Fatalf("run returned %d, want 2", code)
	}
	want := "payments-api\n  overwrite mirror github (destructive)\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}
