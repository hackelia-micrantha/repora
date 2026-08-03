package main

import (
	"bytes"
	"strings"
	"testing"

	"repoctl/internal/apply"
	"repoctl/internal/config"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

func TestRunRoutesMultiMirrorApplyDryRunToAuditedPreflight(t *testing.T) {
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
			ID:  repo.ID,
			UID: repo.DurableID(),
			Mirrors: []status.MirrorResult{
				{Target: "github:org/payments-api", State: status.StateBehind},
				{Target: "gitlab:backup/payments-api", State: status.StateEqual},
			},
		}, nil
	}
	oldBuild := repositoryPlanBuild
	repositoryPlanBuild = func(repo config.Repo, observed status.RepositoryResult) (planartifact.Artifact, error) {
		return planartifact.Artifact{
			Version: planartifact.Version,
			Kind:    planartifact.Kind,
			Repositories: []planartifact.Repository{{
				UID: repo.DurableID(),
				ID:  repo.ID,
				Actions: []planartifact.Action{
					planArtifactAction("github", "org/payments-api", "mirror-0", false),
				},
			}},
		}, nil
	}
	oldAudit := newAudit
	newAudit = func(path string) (*apply.Audit, error) {
		if path != configPath {
			t.Fatalf("audit config path = %q, want %q", path, configPath)
		}
		return &apply.Audit{ExecutionID: "run-multi", Writer: cliJournalWriter{}}, nil
	}
	oldPreflight := repositoryPreflightExecute
	called := false
	repositoryPreflightExecute = func(repo config.Repo, observed status.RepositoryResult, artifact planartifact.Artifact, audit apply.Audit) (apply.Result, error) {
		called = true
		if audit.ExecutionID != "run-multi" || len(artifact.Repositories) != 1 || len(artifact.Repositories[0].Actions) != 1 {
			t.Fatalf("audit/artifact = %#v/%#v", audit, artifact)
		}
		return apply.Result{
			ID:     repo.ID,
			UID:    repo.DurableID(),
			State:  status.StateBehind,
			DryRun: true,
			Actions: []apply.Action{{
				Type:   "PUSH_BRANCH",
				Source: "gitlab:org/payments-api/main",
				Target: "github:org/payments-api/main",
			}},
			Journal: &apply.JournalReferences{
				ExecutionID: "run-multi",
				Intent:      ".repora/journal/intent.json",
				Result:      ".repora/journal/result.json",
			},
		}, nil
	}
	t.Cleanup(func() {
		statusCheckAll = oldCheckAll
		repositoryPlanBuild = oldBuild
		newAudit = oldAudit
		repositoryPreflightExecute = oldPreflight
	})

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"apply", "-f", configPath, "--dry-run"})
	})
	if code != 0 || !called {
		t.Fatalf("code/called = %d/%v, want 0/true", code, called)
	}
	for _, want := range []string{"dry-run PUSH_BRANCH", "github:org/payments-api/main", "run-multi"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
		}
	}
}

func TestMultiMirrorDryRunJSONRejectsBeforeObservation(t *testing.T) {
	spec := config.Spec{Repos: []config.Repo{multiMirrorCommandRepo()}}
	oldCheckAll := statusCheckAll
	called := false
	statusCheckAll = func(config.Repo) (status.RepositoryResult, error) {
		called = true
		return status.RepositoryResult{}, nil
	}
	t.Cleanup(func() { statusCheckAll = oldCheckAll })

	var stderr bytes.Buffer
	code := withStderr(t, &stderr, func() int {
		return runMultiMirrorDryRun(spec, 1, true, nil, "repora.yaml", false)
	})
	if code != 1 || called {
		t.Fatalf("code/called = %d/%v, want 1/false", code, called)
	}
	if !strings.Contains(stderr.String(), "per-target apply contract") {
		t.Fatalf("stderr = %q, want contract guidance", stderr.String())
	}
}

func multiMirrorCommandRepo() config.Repo {
	return config.Repo{
		ID:        "payments-api",
		UID:       "repo.org.payments-api",
		Canonical: config.Endpoint{Provider: "gitlab", Path: "org/payments-api"},
		Mirrors: []config.Endpoint{
			{Provider: "github", Path: "org/payments-api"},
			{Provider: "gitlab", Path: "backup/payments-api"},
		},
	}
}
