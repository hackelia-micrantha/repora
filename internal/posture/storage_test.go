package posture

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGitObjectCounts(t *testing.T) {
	good := "count: 3\nsize: 8\nin-pack: 7\npacks: 1\nsize-pack: 16\nprune-packable: 0\ngarbage: 0\nsize-garbage: 0\n"
	values, err := parseGitObjectCounts(good)
	if err != nil || values["size-pack"] != 16 || values["count"] != 3 {
		t.Fatalf("parse git counters: values=%v err=%v", values, err)
	}
	for _, input := range []string{
		"count: 1\n", strings.Replace(good, "count: 3", "count: -3", 1),
		strings.Replace(good, "size-pack: 16", "size-pack: 9223372036854775807", 1),
		strings.Replace(good, "count: 3", "count: bad", 1),
		good + "count: 2\n",
	} {
		if _, err := parseGitObjectCounts(input); err == nil {
			t.Errorf("accepted invalid count-objects output %q", input)
		}
	}
}

func TestStorageInventoryValidation(t *testing.T) {
	inventory := NewStorageInventory("example/project")
	ev := Evidence{Source: "git-local", Reference: "fixture"}
	inventory.Objects.LooseCount = Observed(int64(0), ev)
	inventory.Objects.LooseBytes = Observed(int64(0), ev)
	inventory.Objects.PackedCount = Observed(int64(0), ev)
	inventory.Objects.PackedBytes = Observed(int64(0), ev)
	inventory.Objects.PackCount = Observed(int64(0), ev)
	inventory.GitState.Shallow = Observed(false, ev)
	inventory.GitState.PromisorConfigured = Observed(false, ev)
	if _, err := inventory.Marshal(); err != nil {
		t.Fatalf("valid inventory: %v", err)
	}
	inventory.Objects.LooseBytes = Observed(int64(-1), ev)
	if err := inventory.Validate(); err == nil {
		t.Fatal("negative bytes were accepted")
	}
	inventory.Objects.LooseBytes = Observed(int64(0), ev)
	inventory.Scope = "all_reachable_history"
	if err := inventory.Validate(); err == nil {
		t.Fatal("unsupported complete-history claim was accepted")
	}
}

func TestCollectLocalGitStorageDoesNotReadRemotes(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	path := t.TempDir()
	cmd := exec.Command("git", "init", "-q", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("initialize fixture: %v: %s", err, out)
	}
	inventory, err := CollectLocalGitStorage(context.Background(), path, "example/project")
	if err != nil {
		t.Fatalf("collect storage: %v", err)
	}
	if inventory.Scope != StorageLocalScope || inventory.GitState.Shallow.Value == nil || *inventory.GitState.Shallow.Value {
		t.Fatalf("unexpected local-only scope/state: %+v", inventory)
	}
	if inventory.Objects.PackCount.State != StateObserved || inventory.Objects.PackCount.Value == nil {
		t.Fatalf("pack count not observed: %+v", inventory.Objects.PackCount)
	}
	data, err := inventory.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), path) {
		t.Fatal("local path leaked in evidence")
	}
	var decoded StorageInventory
	if err := json.Unmarshal(data, &decoded); err != nil || decoded.Validate() != nil {
		t.Fatalf("invalid serialized storage artifact: %v", err)
	}
	// The configured URL must never be reached during this read-only path.
	// A malicious remote would fail if any fetch were attempted.
	cmd = exec.Command("git", "-C", path, "remote", "add", "origin", "https://127.0.0.1:1/should-not-contact")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("configure remote: %v: %s", err, out)
	}
	if _, err := CollectLocalGitStorage(context.Background(), path, "example/project"); err != nil {
		t.Fatalf("unexpected remote contact: %v", err)
	}
}

func TestStorageGitReadOverridesCallerGitDirectory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	path := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", path).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	t.Setenv("GIT_DIR", filepath.Join(t.TempDir(), "not-a-git-directory"))
	if _, err := CollectLocalGitStorage(context.Background(), path, "example/project"); err != nil {
		t.Fatalf("inherited GIT_DIR changed the observation target: %v", err)
	}
}

func TestStoragePromisorConfigAndInheritedTraceSuppression(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	path := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", path).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if out, err := exec.Command("git", "-C", path, "config", "--local", "remote.origin.promisor", "true").CombinedOutput(); err != nil {
		t.Fatalf("git config: %v: %s", err, out)
	}
	trace := filepath.Join(t.TempDir(), "unexpected-trace")
	t.Setenv("GIT_TRACE2_EVENT", trace)
	inventory, err := CollectLocalGitStorage(context.Background(), path, "example/project")
	if err != nil {
		t.Fatalf("read-only collect: %v", err)
	}
	if inventory.GitState.PromisorConfigured.Value == nil || !*inventory.GitState.PromisorConfigured.Value {
		t.Fatalf("promisor config was not observed: %+v", inventory.GitState)
	}
	if _, err := os.Stat(trace); !os.IsNotExist(err) {
		t.Fatalf("collector allowed inherited Git tracing to write a file: %v", err)
	}
}

func TestStorageRepositoryIdentityIsRequired(t *testing.T) {
	if _, err := CollectLocalGitStorage(context.Background(), os.TempDir(), "../bad"); err == nil {
		t.Fatal("accepted invalid operator-supplied repository identity")
	}
}
