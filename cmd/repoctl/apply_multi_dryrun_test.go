package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"repoctl/internal/apply"
	"repoctl/internal/config"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

func TestRunRoutesMultiMirrorDryRunToApplyV3(t *testing.T) {
	configPath := writeMultiMirrorCommandConfig(t)
	restore := installPathBoundCommandFakes(t, configPath, false)
	defer restore()

	oldExecute := repositoryArtifactExecute
	called := false
	repositoryArtifactExecute = func(repo config.Repo, observed status.RepositoryResult, artifact planartifact.Artifact, allowForce, dryRun bool, audit apply.Audit) (apply.DetailedResult, error) {
		called = true
		if allowForce || !dryRun || audit.ExecutionID != "run-multi" {
			t.Fatalf("force/dry/audit = %v/%v/%q", allowForce, dryRun, audit.ExecutionID)
		}
		return detailedCommandResult(repo, true, "VALIDATED", ""), nil
	}
	t.Cleanup(func() { repositoryArtifactExecute = oldExecute })

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"apply", "-f", configPath, "--dry-run", "--json"})
	})
	if code != 0 || !called {
		t.Fatalf("code/called = %d/%v, want 0/true", code, called)
	}
	var output apply.DetailedOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, stdout.String())
	}
	if output.Kind != apply.OutputKind || output.Version != apply.DetailedOutputVersion || output.Results[0].Actions[0].Outcome != "VALIDATED" {
		t.Fatalf("output = %#v", output)
	}
}

func TestRunReportsPartialMultiMirrorFailureAndContinuesOutcomes(t *testing.T) {
	configPath := writeMultiMirrorCommandConfig(t)
	restore := installPathBoundCommandFakes(t, configPath, false)
	defer restore()

	oldExecute := repositoryArtifactExecute
	repositoryArtifactExecute = func(repo config.Repo, observed status.RepositoryResult, artifact planartifact.Artifact, allowForce, dryRun bool, audit apply.Audit) (apply.DetailedResult, error) {
		result := detailedCommandResult(repo, false, "FAILED", "second mirror unavailable")
		result.Actions = append([]apply.DetailedAction{
			{
				Type: "PUSH_BRANCH", Source: "gitlab:org/payments-api/main", Target: "github:org/payments-api/main",
				Before: strings.Repeat("2", 40), Desired: strings.Repeat("1", 40), After: strings.Repeat("1", 40), Outcome: "APPLIED",
			},
		}, result.Actions...)
		return result, errors.New("one mirror action failed")
	}
	t.Cleanup(func() { repositoryArtifactExecute = oldExecute })

	var stdout, stderr bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return withStderr(t, &stderr, func() int {
			return run([]string{"apply", "-f", configPath, "--json"})
		})
	})
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	var output apply.DetailedOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, stdout.String())
	}
	if len(output.Results[0].Actions) != 2 || output.Results[0].Actions[0].Outcome != "APPLIED" || output.Results[0].Actions[1].Outcome != "FAILED" {
		t.Fatalf("actions = %#v, want applied/failed", output.Results[0].Actions)
	}
	if !strings.Contains(stderr.String(), "failed during apply") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunRequiresForceBeforeAuditOrExecution(t *testing.T) {
	configPath := writeMultiMirrorCommandConfig(t)
	restore := installPathBoundCommandFakes(t, configPath, true)
	defer restore()

	oldAudit := newAudit
	auditCalled := false
	newAudit = func(string) (*apply.Audit, error) {
		auditCalled = true
		return nil, errors.New("unexpected audit")
	}
	oldExecute := repositoryArtifactExecute
	executeCalled := false
	repositoryArtifactExecute = func(config.Repo, status.RepositoryResult, planartifact.Artifact, bool, bool, apply.Audit) (apply.DetailedResult, error) {
		executeCalled = true
		return apply.DetailedResult{}, nil
	}
	t.Cleanup(func() {
		newAudit = oldAudit
		repositoryArtifactExecute = oldExecute
	})

	var stderr bytes.Buffer
	code := withStderr(t, &stderr, func() int {
		return run([]string{"apply", "-f", configPath})
	})
	if code != 2 || auditCalled || executeCalled {
		t.Fatalf("code/audit/execute = %d/%v/%v, want 2/false/false", code, auditCalled, executeCalled)
	}
	if !strings.Contains(stderr.String(), "--force") {
		t.Fatalf("stderr = %q, want force guidance", stderr.String())
	}
}

func installPathBoundCommandFakes(t *testing.T, configPath string, force bool) func() {
	t.Helper()
	oldCheckAll := statusCheckAll
	statusCheckAll = func(repo config.Repo) (status.RepositoryResult, error) {
		state := status.StateBehind
		if force {
			state = status.StateDiverged
		}
		return status.RepositoryResult{
			ID: repo.ID, UID: repo.DurableID(),
			Mirrors: []status.MirrorResult{
				{Target: "github:org/payments-api", State: state},
				{Target: "gitlab:backup/payments-api", State: status.StateEqual},
			},
		}, nil
	}
	oldBuild := repositoryPlanBuild
	repositoryPlanBuild = func(repo config.Repo, observed status.RepositoryResult) (planartifact.Artifact, error) {
		action := planArtifactAction("github", "org/payments-api", "mirror-0", force)
		return planartifact.Artifact{
			Version: planartifact.Version,
			Kind:    planartifact.Kind,
			Repositories: []planartifact.Repository{{UID: repo.DurableID(), ID: repo.ID, Actions: []planartifact.Action{action}}},
		}, nil
	}
	oldAudit := newAudit
	newAudit = func(path string) (*apply.Audit, error) {
		if path != configPath {
			t.Fatalf("audit path = %q, want %q", path, configPath)
		}
		return &apply.Audit{ExecutionID: "run-multi", Writer: cliJournalWriter{}}, nil
	}
	return func() {
		statusCheckAll = oldCheckAll
		repositoryPlanBuild = oldBuild
		newAudit = oldAudit
	}
}

func detailedCommandResult(repo config.Repo, dryRun bool, outcome, diagnostic string) apply.DetailedResult {
	return apply.DetailedResult{
		ID: repo.ID, UID: repo.DurableID(), State: status.StateBehind, DryRun: dryRun,
		Actions: []apply.DetailedAction{{
			Type: "PUSH_BRANCH", Source: "gitlab:org/payments-api/main", Target: "gitlab:backup/payments-api/main",
			Before: strings.Repeat("2", 40), Desired: strings.Repeat("1", 40), Outcome: outcome, Error: diagnostic,
		}},
		Journal: &apply.JournalReferences{ExecutionID: "run-multi", Intent: ".repora/journal/intent.json", Result: ".repora/journal/result.json"},
	}
}

func writeMultiMirrorCommandConfig(t *testing.T) string {
	t.Helper()
	return writeConfig(t, `repos:
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
}
