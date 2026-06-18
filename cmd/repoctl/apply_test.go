package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"repoctl/internal/apply"
	"repoctl/internal/config"
	"repoctl/internal/status"
)

func TestApplyJSONIncludesPushBranchActionForBehindMirrorDryRun(t *testing.T) {
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
	oldApply := applyExecute
	statusCheck = func(repo config.Repo) (status.Result, error) {
		return status.Result{ID: repo.ID, State: status.StateBehind, Behind: 2}, nil
	}
	applyExecute = func(repo config.Repo, result status.Result, force, dryRun bool) (apply.Result, error) {
		return apply.Result{
			ID:     repo.ID,
			State:  result.State,
			DryRun: dryRun,
			Actions: []apply.Action{{
				Type:   "PUSH_BRANCH",
				Source: "canonical/main",
				Target: "github/main",
				Force:  false,
			}},
		}, nil
	}
	t.Cleanup(func() {
		statusCheck = oldCheck
		applyExecute = oldApply
	})

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"apply", "-f", configPath, "--json", "--dry-run"})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}

	var got struct {
		Results []struct {
			ID      string `json:"id"`
			State   string `json:"state"`
			Applied bool   `json:"applied"`
			DryRun  bool   `json:"dry_run"`
			Actions []struct {
				Type   string `json:"type"`
				Source string `json:"source"`
				Target string `json:"target"`
				Force  bool   `json:"force"`
			} `json:"actions"`
		} `json:"results"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal json: %v\n%s", err, stdout.String())
	}
	if len(got.Results) != 1 {
		t.Fatalf("results count = %d, want 1", len(got.Results))
	}
	result := got.Results[0]
	if result.ID != "payments-api" || result.State != string(status.StateBehind) || result.Applied || !result.DryRun {
		t.Fatalf("unexpected apply result: %#v", result)
	}
	if len(result.Actions) != 1 {
		t.Fatalf("action count = %d, want 1", len(result.Actions))
	}
	action := result.Actions[0]
	if action.Type != "PUSH_BRANCH" || action.Source != "canonical/main" || action.Target != "github/main" || action.Force {
		t.Fatalf("unexpected action: %#v", action)
	}
}

func TestApplyRequiresForceForDivergedMirror(t *testing.T) {
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
	oldApply := applyExecute
	statusCheck = func(repo config.Repo) (status.Result, error) {
		return status.Result{ID: repo.ID, State: status.StateDiverged, Behind: 1, Ahead: 1}, nil
	}
	applyExecute = func(repo config.Repo, result status.Result, force, dryRun bool) (apply.Result, error) {
		// This should not be called without --force
		return apply.Result{}, nil
	}
	t.Cleanup(func() {
		statusCheck = oldCheck
		applyExecute = oldApply
	})

	var stderr bytes.Buffer
	code := withStderr(t, &stderr, func() int {
		return run([]string{"apply", "-f", configPath})
	})
	if code != 2 {
		t.Fatalf("run returned %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "mirror state is DIVERGED") {
		t.Fatalf("stderr missing divergence error:\n%s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "--force") {
		t.Fatalf("stderr missing --force guidance:\n%s", stderr.String())
	}
}
