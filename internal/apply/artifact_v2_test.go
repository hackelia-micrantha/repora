package apply

import (
	"strings"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

func TestExecuteArtifactV2RejectsTargetPathMismatchBeforeGitReads(t *testing.T) {
	git := &fakeGit{}
	repo := pathRepo()
	planned := pathPlan(repo)
	planned.Actions[0].Target.Path = "other/payments-api"
	artifact, err := planartifact.FromCurrentPlans(planned)
	if err != nil {
		t.Fatalf("FromCurrentPlans returned error: %v", err)
	}

	_, err = ExecuteArtifact(repo, status.Result{State: status.StateBehind}, artifact, git, false, false)
	if err == nil || !strings.Contains(err.Error(), "target path") {
		t.Fatalf("error = %v, want target path mismatch", err)
	}
	assertNoMutation(t, git)
	if len(git.resolveRemoteHeadBranchCalls) != 0 || len(git.resolveRevisionCalls) != 0 {
		t.Fatalf("git reads = heads %#v refs %#v, want topology rejection first", git.resolveRemoteHeadBranchCalls, git.resolveRevisionCalls)
	}
}

func TestExecuteArtifactV1RetainsSingleMirrorAliasCompatibility(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	planned := exactPlan(repo, "main")
	artifact := planartifact.FromLegacyPlans(planned)

	got, err := ExecuteArtifact(repo, status.Result{State: status.StateBehind}, artifact, git, false, true)
	if err != nil {
		t.Fatalf("ExecuteArtifact returned error: %v", err)
	}
	if artifact.Version != planartifact.LegacyVersion || len(got.Actions) != 1 || !got.DryRun {
		t.Fatalf("artifact/result = v%d %#v, want accepted v1 dry-run", artifact.Version, got)
	}
	assertNoMutation(t, git)
}

func pathRepo() config.Repo {
	return config.Repo{
		ID:        "payments-api",
		UID:       "repo.org.payments-api",
		Canonical: config.Endpoint{Provider: "gitlab", Path: "org/payments-api"},
		Mirrors:   []config.Endpoint{{Provider: "github", Path: "org/payments-api"}},
		Mode:      "mirror",
	}
}

func pathPlan(repo config.Repo) plan.ReconciliationPlan {
	return plan.ReconciliationPlan{
		ID:  repo.ID,
		UID: repo.DurableID(),
		Actions: []plan.PlannedAction{{
			Type:              plan.ActionPushBranch,
			Source:            plan.Remote{Provider: "gitlab", Path: "org/payments-api", Name: "canonical", Branch: "main"},
			Target:            plan.Remote{Provider: "github", Path: "org/payments-api", Name: "mirror", Branch: "main"},
			ExpectedSource:    testOID,
			ExpectedOldTarget: testOID,
			Reason:            "mirror is behind",
		}},
	}
}
