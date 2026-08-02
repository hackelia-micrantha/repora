package main

import (
	"bytes"
	"testing"

	"repoctl/internal/apply"
	"repoctl/internal/config"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

func TestApplyPlanFileAllowsForcedDryRunWithoutAuthorization(t *testing.T) {
	configPath := writeConfig(t, `repos:
  - id: payments-api
    uid: repo.org.payments-api
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/payments-api.git
    mirrors:
      - provider: github
        url: git@github.com:org/payments-api.git
`)
	artifact := behindArtifact(t)
	artifact.Repositories[0].Actions[0].Force = true
	artifactPath := writeArtifact(t, artifact)

	oldCheck := statusCheck
	statusCheck = func(repo config.Repo) (status.Result, error) {
		return status.Result{ID: repo.ID, UID: repo.DurableID(), State: status.StateDiverged}, nil
	}
	oldArtifactApply := artifactApplyExecute
	called := false
	artifactApplyExecute = func(repo config.Repo, result status.Result, got planartifact.Artifact, force, dryRun bool) (apply.Result, error) {
		called = true
		if force || !dryRun {
			t.Fatalf("force/dryRun = %v/%v, want false/true", force, dryRun)
		}
		return apply.Result{
			ID:     repo.ID,
			UID:    repo.DurableID(),
			State:  result.State,
			DryRun: true,
			Actions: []apply.Action{{
				Type:              "PUSH_BRANCH",
				Source:            "canonical/main",
				Target:            "github/main",
				Force:             true,
				ExpectedOldTarget: testTargetOID,
			}},
		}, nil
	}
	t.Cleanup(func() {
		statusCheck = oldCheck
		artifactApplyExecute = oldArtifactApply
	})

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"apply", "-f", configPath, "--plan-file", artifactPath, "--dry-run", "--json"})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if !called {
		t.Fatal("artifactApplyExecute was not called")
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"force":true`)) {
		t.Fatalf("stdout = %q, want forced dry-run action", stdout.String())
	}
}
