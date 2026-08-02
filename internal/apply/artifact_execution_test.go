package apply

import (
	"strings"
	"testing"

	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

func TestExecuteArtifactUsesProvidedActionWithoutReplanning(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	planned := plan.ReconciliationPlan{
		ID:  repo.ID,
		UID: repo.DurableID(),
		Actions: []plan.PlannedAction{{
			Type:              plan.ActionPushBranch,
			Source:            plan.Remote{Provider: "gitlab", Name: "canonical", Branch: "release"},
			Target:            plan.Remote{Provider: "github", Name: "mirror", Branch: "release"},
			ExpectedSource:    testOID,
			ExpectedOldTarget: testOID,
			Reason:            "mirror is behind",
		}},
	}
	artifact := planartifact.FromPlans(planned)

	got, err := ExecuteArtifact(repo, status.Result{State: status.StateBehind}, artifact, git, false, false)
	if err != nil {
		t.Fatalf("ExecuteArtifact returned error: %v", err)
	}
	if len(git.resolveRemoteHeadBranchCalls) != 0 {
		t.Fatalf("remote HEAD calls = %#v, want no replanning", git.resolveRemoteHeadBranchCalls)
	}
	if len(git.pushBranchCalls) != 1 {
		t.Fatalf("push calls = %#v, want one exact artifact push", git.pushBranchCalls)
	}
	call := git.pushBranchCalls[0]
	if call.srcRef != "refs/remotes/canonical/release" || call.dstBranch != "release" {
		t.Fatalf("push call = %#v, want artifact branches", call)
	}
	if len(got.Actions) != 1 || got.Actions[0].Source != "canonical/release" || got.Actions[0].Target != "github/release" {
		t.Fatalf("result actions = %#v, want artifact-derived compatibility action", got.Actions)
	}
}

func TestExecuteArtifactRejectsTopologyMismatchBeforeMutation(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	planned := plan.ReconciliationPlan{
		ID:  repo.ID,
		UID: repo.DurableID(),
		Actions: []plan.PlannedAction{{
			Type:              plan.ActionPushBranch,
			Source:            plan.Remote{Provider: "gitlab", Name: "canonical", Branch: "main"},
			Target:            plan.Remote{Provider: "gitlab", Name: "mirror", Branch: "main"},
			ExpectedSource:    testOID,
			ExpectedOldTarget: testOID,
			Reason:            "mirror is behind",
		}},
	}

	_, err := ExecuteArtifact(repo, status.Result{State: status.StateBehind}, planartifact.FromPlans(planned), git, false, false)
	if err == nil || !strings.Contains(err.Error(), "does not match configured mirror") {
		t.Fatalf("error = %v, want topology mismatch", err)
	}
	assertNoMutation(t, git)
	if len(git.resolveRevisionCalls) != 0 {
		t.Fatalf("resolve calls = %#v, want rejection before stale preflight", git.resolveRevisionCalls)
	}
}

func TestExecuteArtifactRequiresForceAuthorization(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	planned := plan.ReconciliationPlan{
		ID:  repo.ID,
		UID: repo.DurableID(),
		Actions: []plan.PlannedAction{{
			Type:              plan.ActionPushBranch,
			Source:            plan.Remote{Provider: "gitlab", Name: "canonical", Branch: "main"},
			Target:            plan.Remote{Provider: "github", Name: "mirror", Branch: "main"},
			ExpectedSource:    testOID,
			ExpectedOldTarget: testOID,
			Force:             true,
			Reason:            "mirror is diverged",
		}},
	}

	_, err := ExecuteArtifact(repo, status.Result{State: status.StateDiverged}, planartifact.FromPlans(planned), git, false, false)
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("error = %v, want force authorization failure", err)
	}
	assertNoMutation(t, git)
}

func TestExecuteArtifactMatchesDurableUIDAcrossAliasChange(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	repo.UID = "repo.org.payments-api"
	planned := plan.ReconciliationPlan{ID: "old-payments-name", UID: repo.UID, Actions: []plan.PlannedAction{}}

	got, err := ExecuteArtifact(repo, status.Result{State: status.StateEqual}, planartifact.FromPlans(planned), git, false, true)
	if err != nil {
		t.Fatalf("ExecuteArtifact returned error: %v", err)
	}
	if got.ID != repo.ID || got.UID != repo.UID {
		t.Fatalf("result identity = %q/%q, want current alias and durable uid", got.ID, got.UID)
	}
}
