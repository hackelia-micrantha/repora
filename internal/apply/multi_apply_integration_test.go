package apply_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"repoctl/internal/apply"
	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/journal"
	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

func TestMultiMirrorApplyContinuesAfterBareRemoteFailure(t *testing.T) {
	requireIntegration(t)
	root := t.TempDir()
	setHomeEnv(t, root)

	canonicalBare := filepath.Join(root, "canonical.git")
	canonicalWork := filepath.Join(root, "canonical-work")
	mirrors := []string{
		filepath.Join(root, "mirror-zero.git"),
		filepath.Join(root, "mirror-one.git"),
		filepath.Join(root, "mirror-two.git"),
	}
	git(t, root, "init", "--bare", canonicalBare)
	git(t, canonicalBare, "symbolic-ref", "HEAD", "refs/heads/main")
	for _, mirror := range mirrors {
		git(t, root, "init", "--bare", mirror)
		git(t, mirror, "symbolic-ref", "HEAD", "refs/heads/main")
	}

	git(t, root, "clone", canonicalBare, canonicalWork)
	git(t, canonicalWork, "config", "user.name", "Repora Test")
	git(t, canonicalWork, "config", "user.email", "repora@example.com")
	writeFile(t, filepath.Join(canonicalWork, "README.md"), "v1\n")
	git(t, canonicalWork, "add", "README.md")
	git(t, canonicalWork, "commit", "-m", "initial")
	git(t, canonicalWork, "branch", "-M", "main")
	git(t, canonicalWork, "push", "origin", "main")
	for _, mirror := range mirrors {
		git(t, canonicalWork, "push", mirror, "main")
	}

	writeFile(t, filepath.Join(canonicalWork, "README.md"), "v2\n")
	git(t, canonicalWork, "add", "README.md")
	git(t, canonicalWork, "commit", "-m", "canonical ahead")
	git(t, canonicalWork, "push", "origin", "main")

	repo := config.Repo{
		ID:        "payments-api",
		UID:       "repo.org.payments-api",
		Canonical: config.Endpoint{Provider: "gitlab", Path: "org/payments-api"},
		Mirrors: []config.Endpoint{
			{Provider: "github", Path: "zero/payments-api"},
			{Provider: "gitlab", Path: "one/payments-api"},
			{Provider: "github", Path: "two/payments-api"},
		},
		Mode: "mirror",
	}
	cachePath, err := gitwrap.MirrorPath(repo.DurableID())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o700); err != nil {
		t.Fatal(err)
	}
	git(t, root, "clone", "--mirror", canonicalBare, cachePath)
	git(t, cachePath, "remote", "rename", "origin", "canonical")
	git(t, cachePath, "fetch", "canonical", "+refs/heads/*:refs/remotes/canonical/*")
	git(t, cachePath, "remote", "set-head", "canonical", "main")
	for i, mirror := range mirrors {
		remote := fmt.Sprintf("mirror-%d", i)
		git(t, cachePath, "remote", "add", remote, mirror)
		git(t, cachePath, "fetch", remote)
		git(t, cachePath, "remote", "set-head", remote, "main")
	}

	sourceOID := strings.TrimSpace(git(t, cachePath, "rev-parse", "refs/remotes/canonical/main"))
	actions := make([]plan.PlannedAction, 0, len(repo.Mirrors))
	observed := status.RepositoryResult{ID: repo.ID, UID: repo.DurableID(), Mirrors: make([]status.MirrorResult, 0, len(repo.Mirrors))}
	for i, mirror := range repo.Mirrors {
		remote := fmt.Sprintf("mirror-%d", i)
		targetOID := strings.TrimSpace(git(t, cachePath, "rev-parse", "refs/remotes/"+remote+"/main"))
		actions = append(actions, plan.PlannedAction{
			Type:           plan.ActionPushBranch,
			Source:         plan.Remote{Provider: repo.Canonical.Provider, Path: repo.Canonical.Path, Name: "canonical", Branch: "main"},
			Target:         plan.Remote{Provider: mirror.Provider, Path: mirror.Path, Name: remote, Branch: "main"},
			ExpectedSource: sourceOID, ExpectedOldTarget: targetOID, Reason: "mirror is behind",
		})
		target, err := mirror.TargetID()
		if err != nil {
			t.Fatal(err)
		}
		observed.Mirrors = append(observed.Mirrors, status.MirrorResult{Target: target, State: status.StateBehind})
	}
	artifact, err := planartifact.FromCurrentPlans(plan.ReconciliationPlan{ID: repo.ID, UID: repo.DurableID(), Actions: actions})
	if err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(mirrors[1]); err != nil {
		t.Fatal(err)
	}
	result, err := apply.ExecuteRepositoryArtifactAudited(repo, observed, artifact, gitwrap.Client{}, false, false, apply.Audit{
		ExecutionID: "run-partial-integration",
		Writer:      journal.Writer{Root: root},
	})
	if err == nil || !strings.Contains(err.Error(), "one/payments-api") {
		t.Fatalf("error = %v, want middle target failure", err)
	}
	if len(result.Actions) != 3 || result.Actions[0].Outcome != "APPLIED" || result.Actions[1].Outcome != "FAILED" || result.Actions[2].Outcome != "APPLIED" {
		t.Fatalf("actions = %#v, want applied/failed/applied", result.Actions)
	}
	for _, index := range []int{0, 2} {
		got := strings.TrimSpace(git(t, mirrors[index], "rev-parse", "refs/heads/main"))
		if got != sourceOID {
			t.Fatalf("mirror %d head = %s, want %s", index, got, sourceOID)
		}
	}
	if result.Journal == nil || result.Journal.Intent == "" || result.Journal.Result == "" {
		t.Fatalf("journal = %#v, want intent/result evidence", result.Journal)
	}
}
