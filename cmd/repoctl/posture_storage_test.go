package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"testing"

	"repoctl/internal/posture"
)

func TestPostureStorageCLIProducesScopedJSON(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	path := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", path).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"posture", "storage", "--repository", "example/project", "--path", path})
	})
	if code != 0 {
		t.Fatalf("storage CLI failed: code=%d output=%s", code, stdout.String())
	}
	var inventory posture.StorageInventory
	if err := json.Unmarshal(stdout.Bytes(), &inventory); err != nil {
		t.Fatal(err)
	}
	if err := inventory.Validate(); err != nil {
		t.Fatalf("invalid storage contract: %v", err)
	}
	if inventory.Scope != posture.StorageLocalScope {
		t.Fatalf("unexpected scope %q", inventory.Scope)
	}
}

func TestPostureStorageCLIRequiresRepositoryAndPath(t *testing.T) {
	var stderr bytes.Buffer
	code := withStderr(t, &stderr, func() int {
		return run([]string{"posture", "storage", "--repository", "example/project"})
	})
	if code != 1 {
		t.Fatalf("missing explicit local path accepted: code=%d", code)
	}
}
