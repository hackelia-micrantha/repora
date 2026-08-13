package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/managedartifact"
)

func TestApplyREADMEDryRunPrintsReviewedPlan(t *testing.T) {
	configPath := writeManagedPlanConfig(t)
	plan := managedPlanFixture(t)
	planPath := writeManagedPlanFile(t, plan)
	stubManagedPreflight(t, nil)

	var stdout bytes.Buffer
	code := captureManagedStdout(t, &stdout, func() int {
		return run([]string{"apply-readme", "-f", configPath, "--plan-file", planPath, "--dry-run"})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), plan.Repositories[0].Actions[0].Diff) {
		t.Fatalf("stdout missing reviewed diff:\n%s", stdout.String())
	}
}

func TestApplyREADMEDryRunReturnsTwoForStalePlan(t *testing.T) {
	configPath := writeManagedPlanConfig(t)
	planPath := writeManagedPlanFile(t, managedPlanFixture(t))
	stubManagedPreflight(t, fmt.Errorf("%w: canonical HEAD changed", managedartifact.ErrStale))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := captureManagedStdout(t, &stdout, func() int {
		return captureManagedStderr(t, &stderr, func() int {
			return run([]string{"apply-readme", "-f", configPath, "--plan-file", planPath, "--dry-run"})
		})
	})
	if code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "managed artifact plan is stale") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestApplyREADMEDryRunReturnsOneForObservationFailure(t *testing.T) {
	configPath := writeManagedPlanConfig(t)
	planPath := writeManagedPlanFile(t, managedPlanFixture(t))
	stubManagedPreflight(t, errors.New("fetch unavailable"))

	var stderr bytes.Buffer
	code := captureManagedStderr(t, &stderr, func() int {
		return run([]string{"apply-readme", "-f", configPath, "--plan-file", planPath, "--dry-run"})
	})
	if code != 1 || !strings.Contains(stderr.String(), "fetch unavailable") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}

func TestApplyREADMERequiresDryRunBeforeIO(t *testing.T) {
	var stderr bytes.Buffer
	code := captureManagedStderr(t, &stderr, func() int {
		return run([]string{"apply-readme", "--plan-file", "/does/not/exist"})
	})
	if code != 1 || !strings.Contains(stderr.String(), "currently requires --dry-run") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}

func TestApplyREADMEDryRunRequiresPlanFile(t *testing.T) {
	var stderr bytes.Buffer
	code := captureManagedStderr(t, &stderr, func() int {
		return run([]string{"apply-readme", "--dry-run"})
	})
	if code != 1 || !strings.Contains(stderr.String(), "requires --plan-file FILE") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}

func TestApplyREADMEHelp(t *testing.T) {
	var stdout bytes.Buffer
	code := captureManagedStdout(t, &stdout, func() int {
		return run([]string{"apply-readme", "--help"})
	})
	if code != 0 || !strings.Contains(stdout.String(), "usage: repoctl apply-readme") {
		t.Fatalf("code=%d stdout=%q", code, stdout.String())
	}
}

func stubManagedPreflight(t *testing.T, preflightErr error) {
	t.Helper()
	oldPreflight := managedREADMEPreflight
	oldObserver := managedREADMEPreflightObserver
	managedREADMEPreflight = func(config.Spec, managedartifact.Plan, managedartifact.READMEObserver) error {
		return preflightErr
	}
	managedREADMEPreflightObserver = func() managedartifact.READMEObserver { return nil }
	t.Cleanup(func() {
		managedREADMEPreflight = oldPreflight
		managedREADMEPreflightObserver = oldObserver
	})
}

func writeManagedPlanFile(t *testing.T, plan managedartifact.Plan) string {
	t.Helper()
	data, err := plan.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "managed-plan.json")
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
