package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"repoctl/internal/apply"
	"repoctl/internal/config"
	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

const (
	testSourceOID = "1111111111111111111111111111111111111111"
	testTargetOID = "2222222222222222222222222222222222222222"
)

func TestMain(m *testing.M) {
	oldPlanBuild := planBuild
	planBuild = func(repo config.Repo, result status.Result, force bool) (planartifact.Artifact, error) {
		planned, err := plan.Reconcile(repo, result, plan.Observation{
			CanonicalBranch:  "main",
			CanonicalHeadOID: testSourceOID,
			MirrorBranch:     "main",
			MirrorHeadOID:    testTargetOID,
		}, true)
		if err != nil {
			return planartifact.Artifact{}, err
		}
		return planartifact.FromPlans(planned), nil
	}
	code := m.Run()
	planBuild = oldPlanBuild
	os.Exit(code)
}

func TestPlanArtifactEmitsExactExecutableArtifact(t *testing.T) {
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

	oldCheck := statusCheck
	statusCheck = func(repo config.Repo) (status.Result, error) {
		return status.Result{ID: repo.ID, UID: repo.DurableID(), State: status.StateBehind, Behind: 3}, nil
	}
	t.Cleanup(func() { statusCheck = oldCheck })

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"plan", "-f", configPath, "--artifact"})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	artifact, err := planartifact.Parse(stdout.Bytes())
	if err != nil {
		t.Fatalf("parse artifact: %v\n%s", err, stdout.String())
	}
	if len(artifact.Repositories) != 1 || len(artifact.Repositories[0].Actions) != 1 {
		t.Fatalf("artifact = %#v, want one repository action", artifact)
	}
	action := artifact.Repositories[0].Actions[0]
	if action.Type != "PUSH_BRANCH" || action.Source.Remote != "canonical" || action.Target.Remote != "mirror" {
		t.Fatalf("action = %#v, want exact branch action", action)
	}
	if action.Diff.Observed != testTargetOID || action.Diff.Desired != testSourceOID {
		t.Fatalf("diff = %#v, want observed and desired OIDs", action.Diff)
	}
}

func TestPlanArtifactIsNotEmittedWhenPlanningIsIncomplete(t *testing.T) {
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
		return status.Result{ID: repo.ID, UID: repo.DurableID(), State: status.StateBehind}, nil
	}
	oldBuild := planBuild
	planBuild = func(repo config.Repo, result status.Result, force bool) (planartifact.Artifact, error) {
		return planartifact.Artifact{}, errors.New("resolve canonical branch")
	}
	t.Cleanup(func() {
		statusCheck = oldCheck
		planBuild = oldBuild
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return withStderr(t, &stderr, func() int {
			return run([]string{"plan", "-f", configPath, "--artifact"})
		})
	})
	if code != 1 {
		t.Fatalf("run returned %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want no partial executable artifact", stdout.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("exact plan artifact not emitted")) {
		t.Fatalf("stderr = %q, want incomplete planning diagnostic", stderr.String())
	}
}

func TestApplyPlanFileConsumesExactArtifactWithoutReplanning(t *testing.T) {
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
	artifactPath := writeArtifact(t, artifact)

	oldCheck := statusCheck
	statusCheck = func(repo config.Repo) (status.Result, error) {
		return status.Result{ID: repo.ID, UID: repo.DurableID(), State: status.StateBehind}, nil
	}
	oldBuild := planBuild
	planBuild = func(repo config.Repo, result status.Result, force bool) (planartifact.Artifact, error) {
		t.Fatal("planBuild called while applying an exact artifact")
		return planartifact.Artifact{}, nil
	}
	oldArtifactApply := artifactApplyExecute
	called := false
	artifactApplyExecute = func(repo config.Repo, result status.Result, got planartifact.Artifact, force, dryRun bool) (apply.Result, error) {
		called = true
		if got.Repositories[0].Actions[0].Diff.Desired != testSourceOID {
			t.Fatalf("executed artifact = %#v, want original desired OID", got)
		}
		return apply.Result{ID: repo.ID, UID: repo.DurableID(), State: result.State, DryRun: dryRun, Actions: []apply.Action{{Type: "PUSH_BRANCH", Source: "canonical/main", Target: "github/main"}}}, nil
	}
	t.Cleanup(func() {
		statusCheck = oldCheck
		planBuild = oldBuild
		artifactApplyExecute = oldArtifactApply
	})

	code := run([]string{"apply", "-f", configPath, "--plan-file", artifactPath, "--dry-run", "--json"})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if !called {
		t.Fatal("artifactApplyExecute was not called")
	}
}

func TestApplyPlanFileRequiresExplicitForceAuthorization(t *testing.T) {
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
	artifactApplyExecute = func(repo config.Repo, result status.Result, got planartifact.Artifact, force, dryRun bool) (apply.Result, error) {
		t.Fatal("artifactApplyExecute called without force authorization")
		return apply.Result{}, nil
	}
	t.Cleanup(func() {
		statusCheck = oldCheck
		artifactApplyExecute = oldArtifactApply
	})

	var stderr bytes.Buffer
	code := withStderr(t, &stderr, func() int {
		return run([]string{"apply", "-f", configPath, "--plan-file", artifactPath})
	})
	if code != 2 {
		t.Fatalf("run returned %d, want 2", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("contains a forced action")) {
		t.Fatalf("stderr = %q, want force guidance", stderr.String())
	}
}

func behindArtifact(t *testing.T) planartifact.Artifact {
	t.Helper()
	repo := config.Repo{
		ID:        "payments-api",
		UID:       "repo.org.payments-api",
		Canonical: config.Endpoint{Provider: "gitlab"},
		Mirrors:   []config.Endpoint{{Provider: "github"}},
	}
	planned, err := plan.Reconcile(repo, status.Result{State: status.StateBehind}, plan.Observation{
		CanonicalBranch:  "main",
		CanonicalHeadOID: testSourceOID,
		MirrorBranch:     "main",
		MirrorHeadOID:    testTargetOID,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	return planartifact.FromPlans(planned)
}

func writeArtifact(t *testing.T, artifact planartifact.Artifact) string {
	t.Helper()
	data, err := artifact.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
