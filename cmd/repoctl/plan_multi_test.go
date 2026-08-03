package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

func TestMultiMirrorPlanArtifactIncludesEveryRequiredTarget(t *testing.T) {
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
			ID: repo.ID, UID: repo.DurableID(),
			Mirrors: []status.MirrorResult{
				{Target: "gitlab:backup/payments-api", Provider: "gitlab", Path: "backup/payments-api", State: status.StateBehind},
				{Target: "github:org/payments-api", Provider: "github", Path: "org/payments-api", State: status.StateAhead},
			},
		}, nil
	}
	oldBuild := repositoryPlanBuild
	repositoryPlanBuild = func(repo config.Repo, observed status.RepositoryResult) (planartifact.Artifact, error) {
		return planartifact.Artifact{
			Version: planartifact.Version,
			Kind:    planartifact.Kind,
			Repositories: []planartifact.Repository{{
				UID: repo.DurableID(), ID: repo.ID,
				Actions: []planartifact.Action{
					planArtifactAction("github", "org/payments-api", "mirror-0", false),
					planArtifactAction("gitlab", "backup/payments-api", "mirror-1", true),
				},
			}},
		}, nil
	}
	t.Cleanup(func() {
		statusCheckAll = oldCheckAll
		repositoryPlanBuild = oldBuild
	})

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"plan", "-f", configPath, "--artifact"})
	})
	if code != 2 {
		t.Fatalf("run returned %d, want 2 for reviewed destructive intent", code)
	}
	var artifact planartifact.Artifact
	if err := json.Unmarshal(stdout.Bytes(), &artifact); err != nil {
		t.Fatalf("unmarshal artifact: %v\n%s", err, stdout.String())
	}
	if len(artifact.Repositories) != 1 || len(artifact.Repositories[0].Actions) != 2 {
		t.Fatalf("artifact = %#v, want two actions", artifact)
	}
	if artifact.Repositories[0].Actions[0].Target.Path != "org/payments-api" || artifact.Repositories[0].Actions[1].Target.Path != "backup/payments-api" {
		t.Fatalf("targets = %#v", artifact.Repositories[0].Actions)
	}
}

func TestMultiMirrorPlanJSONRejectsBeforeObservation(t *testing.T) {
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
	called := false
	statusCheckAll = func(config.Repo) (status.RepositoryResult, error) {
		called = true
		return status.RepositoryResult{}, nil
	}
	t.Cleanup(func() { statusCheckAll = oldCheckAll })

	var stderr bytes.Buffer
	code := withStderr(t, &stderr, func() int {
		return run([]string{"plan", "-f", configPath, "--json"})
	})
	if code != 1 || called {
		t.Fatalf("code/called = %d/%v, want 1/false", code, called)
	}
	if !strings.Contains(stderr.String(), "use --artifact") {
		t.Fatalf("stderr = %q, want exact artifact guidance", stderr.String())
	}
}

func TestMultiMirrorArtifactSuppressedWhenObservationIncomplete(t *testing.T) {
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
		return status.RepositoryResult{ID: repo.ID}, errors.New("mirror unavailable")
	}
	t.Cleanup(func() { statusCheckAll = oldCheckAll })

	var stdout, stderr bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return withStderr(t, &stderr, func() int {
			return run([]string{"plan", "-f", configPath, "--artifact"})
		})
	})
	if code != 1 || stdout.Len() != 0 {
		t.Fatalf("code/stdout = %d/%q, want 1/empty", code, stdout.String())
	}
	if !strings.Contains(stderr.String(), "not emitted") {
		t.Fatalf("stderr = %q, want artifact suppression", stderr.String())
	}
}

func planArtifactAction(provider, path, remote string, force bool) planartifact.Action {
	return planartifact.Action{
		Type: "PUSH_BRANCH",
		Source: planartifact.Ref{Provider: "gitlab", Path: "org/payments-api", Remote: "canonical", Branch: "main"},
		Target: planartifact.Ref{Provider: provider, Path: path, Remote: remote, Branch: "main"},
		Diff: planartifact.RefDiff{
			Observed: "2222222222222222222222222222222222222222",
			Desired:  "1111111111111111111111111111111111111111",
		},
		Force:  force,
		Reason: "mirror requires reconciliation",
	}
}
