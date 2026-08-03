package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/status"
)

func TestStatusJSONPreservesMixedMirrorResults(t *testing.T) {
	configPath := writeConfig(t, `repos:
  - id: payments-api
    uid: repo.org.payments-api
    canonical:
      provider: gitlab
      path: org/payments-api
    mirrors:
      - provider: github
        path: org/payments-api
      - provider: gitlab
        path: backup/payments-api
`)

	oldCheckAll := statusCheckAll
	statusCheckAll = func(repo config.Repo) (status.RepositoryResult, error) {
		return status.RepositoryResult{
			ID:        repo.ID,
			UID:       repo.DurableID(),
			Canonical: status.RefResult{Ref: "HEAD", Commit: "canon123"},
			Mirrors: []status.MirrorResult{
				{Target: "github:org/payments-api", Provider: "github", Path: "org/payments-api", Ref: "HEAD", State: status.StateError, Error: "fetch mirror: unavailable"},
				{Target: "gitlab:backup/payments-api", Provider: "gitlab", Path: "backup/payments-api", Ref: "HEAD", Commit: "backup123", State: status.StateEqual},
			},
		}, errors.New("github:org/payments-api: fetch mirror: unavailable")
	}
	t.Cleanup(func() { statusCheckAll = oldCheckAll })

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"status", "-f", configPath, "--json"})
	})
	if code != 1 {
		t.Fatalf("run returned %d, want 1", code)
	}
	var output status.Output
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("unmarshal status JSON: %v\n%s", err, stdout.String())
	}
	if output.Kind != status.OutputKind || output.Version != status.OutputVersion || len(output.Repos) != 1 || len(output.Repos[0].Mirrors) != 2 {
		t.Fatalf("output = %#v", output)
	}
	if output.Repos[0].Mirrors[0].State != status.StateError || output.Repos[0].Mirrors[1].State != status.StateEqual {
		t.Fatalf("mirrors = %#v", output.Repos[0].Mirrors)
	}
}

func TestStatusReturnsUnsafeCodeOnlyWhenObservationComplete(t *testing.T) {
	configPath := writeConfig(t, `repos:
  - id: payments-api
    canonical:
      provider: gitlab
      path: org/payments-api
    mirrors:
      - provider: github
        path: org/payments-api
      - provider: gitlab
        path: backup/payments-api
`)

	oldCheckAll := statusCheckAll
	statusCheckAll = func(repo config.Repo) (status.RepositoryResult, error) {
		return status.RepositoryResult{
			ID: repo.ID,
			Mirrors: []status.MirrorResult{
				{Target: "github:org/payments-api", Provider: "github", Path: "org/payments-api", Ref: "HEAD", State: status.StateAhead, Ahead: 1},
				{Target: "gitlab:backup/payments-api", Provider: "gitlab", Path: "backup/payments-api", Ref: "HEAD", State: status.StateEqual},
			},
		}, nil
	}
	t.Cleanup(func() { statusCheckAll = oldCheckAll })

	code := withStdout(t, &bytes.Buffer{}, func() int {
		return run([]string{"status", "-f", configPath})
	})
	if code != 2 {
		t.Fatalf("run returned %d, want 2", code)
	}
}

func TestPlanRejectsMultiMirrorBeforeObservation(t *testing.T) {
	configPath := writeConfig(t, `repos:
  - id: payments-api
    canonical:
      provider: gitlab
      path: org/payments-api
    mirrors:
      - provider: github
        path: org/payments-api
      - provider: gitlab
        path: backup/payments-api
`)

	oldCheck := statusCheck
	called := false
	statusCheck = func(config.Repo) (status.Result, error) {
		called = true
		return status.Result{}, nil
	}
	t.Cleanup(func() { statusCheck = oldCheck })

	var stderr bytes.Buffer
	code := withStderr(t, &stderr, func() int {
		return run([]string{"plan", "-f", configPath})
	})
	if code != 1 {
		t.Fatalf("run returned %d, want 1", code)
	}
	if called {
		t.Fatal("status observation ran before the multi-mirror mutation gate")
	}
	if !strings.Contains(stderr.String(), "remain single-mirror") {
		t.Fatalf("stderr = %q, want explicit mutation gate", stderr.String())
	}
}
