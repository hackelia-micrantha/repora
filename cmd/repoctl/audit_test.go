package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"repoctl/internal/apply"
	"repoctl/internal/config"
	"repoctl/internal/journal"
	"repoctl/internal/status"
)

func TestDefaultAuditAnchorsJournalBesideConfiguration(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config", "repora.yaml")
	audit, err := defaultAudit(configPath)
	if err != nil {
		t.Fatalf("defaultAudit returned error: %v", err)
	}
	if !strings.HasPrefix(audit.ExecutionID, "run-") || len(audit.ExecutionID) != len("run-")+32 {
		t.Fatalf("execution ID = %q, want run- plus 128-bit hex entropy", audit.ExecutionID)
	}
	writer, ok := audit.Writer.(journal.Writer)
	if !ok {
		t.Fatalf("writer type = %T, want journal.Writer", audit.Writer)
	}
	wantRoot, err := filepath.Abs(filepath.Dir(configPath))
	if err != nil {
		t.Fatal(err)
	}
	if writer.Root != wantRoot {
		t.Fatalf("journal root = %q, want %q", writer.Root, wantRoot)
	}
}

func TestRunApplyUsesCommandAuditAndEmitsJournalReferences(t *testing.T) {
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
		return status.Result{ID: repo.ID, UID: repo.DurableID(), State: status.StateBehind}, nil
	}
	oldNewAudit := newAudit
	newAudit = func(path string) (*apply.Audit, error) {
		if path != configPath {
			t.Fatalf("audit config path = %q, want %q", path, configPath)
		}
		return &apply.Audit{ExecutionID: "run-cli", Writer: cliJournalWriter{}}, nil
	}
	oldAuditedApply := auditedApplyExecute
	called := false
	auditedApplyExecute = func(repo config.Repo, result status.Result, force, dryRun bool, audit apply.Audit) (apply.Result, error) {
		called = true
		if audit.ExecutionID != "run-cli" || !dryRun {
			t.Fatalf("audit/dry-run = %#v/%v, want run-cli/true", audit, dryRun)
		}
		return apply.Result{
			ID:      repo.ID,
			UID:     repo.DurableID(),
			State:   result.State,
			DryRun:  true,
			Actions: []apply.Action{{Type: "PUSH_BRANCH", Source: "canonical/main", Target: "github/main"}},
			Journal: &apply.JournalReferences{
				ExecutionID: "run-cli",
				Intent:      ".repora/journal/repo.org.payments-api--run-cli--intent.json",
				Result:      ".repora/journal/repo.org.payments-api--run-cli--result.json",
			},
		}, nil
	}
	t.Cleanup(func() {
		statusCheck = oldCheck
		newAudit = oldNewAudit
		auditedApplyExecute = oldAuditedApply
	})

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"apply", "-f", configPath, "--dry-run", "--json"})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if !called {
		t.Fatal("auditedApplyExecute was not called")
	}
	for _, want := range []string{`"version":2`, `"execution_id":"run-cli"`, `"intent":".repora/journal/`, `"result":".repora/journal/`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
		}
	}
}

type cliJournalWriter struct{}

func (cliJournalWriter) Write(record journal.Record) (string, error) {
	return ".repora/journal/test.json", nil
}
