package apply

import (
	"strings"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

func TestExecuteArtifactUsesProvidedActionWithoutReplanning(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	artifact := planartifact.FromPlans(exactPlan(repo, "main"))

	got, err := ExecuteArtifact(repo, status.Result{State: status.StateBehind}, artifact, git, false, false)
	if err != nil {
		t.Fatalf("ExecuteArtifact returned error: %v", err)
	}
	if len(git.resolveRemoteHeadBranchCalls) != 2 || git.resolveRemoteHeadBranchCalls[0] != "canonical" || git.resolveRemoteHeadBranchCalls[1] != "mirror" {
		t.Fatalf("remote HEAD calls = %#v, want scope validation only", git.resolveRemoteHeadBranchCalls)
	}
	if len(git.pushBranchCalls) != 1 {
		t.Fatalf("push calls = %#v, want one exact artifact push", git.pushBranchCalls)
	}
	call := git.pushBranchCalls[0]
	if call.srcRef != "refs/remotes/canonical/main" || call.dstBranch != "main" {
		t.Fatalf("push call = %#v, want artifact branches", call)
	}
	if len(got.Actions) != 1 || got.Actions[0].Source != "canonical/main" || got.Actions[0].Target != "github/main" {
		t.Fatalf("result actions = %#v, want artifact-derived compatibility action", got.Actions)
	}
}

func TestExecuteArtifactRejectsTopologyMismatchBeforeMutation(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	planned := exactPlan(repo, "main")
	planned.Actions[0].Target.Provider = "gitlab"

	_, err := ExecuteArtifact(repo, status.Result{State: status.StateBehind}, planartifact.FromPlans(planned), git, false, false)
	if err == nil || !strings.Contains(err.Error(), "does not match configured mirror") {
		t.Fatalf("error = %v, want topology mismatch", err)
	}
	assertNoMutation(t, git)
	if len(git.resolveRevisionCalls) != 0 || len(git.resolveRemoteHeadBranchCalls) != 0 {
		t.Fatalf("git reads = refs %#v heads %#v, want rejection before repository reads", git.resolveRevisionCalls, git.resolveRemoteHeadBranchCalls)
	}
}

func TestExecuteArtifactRejectsNonDefaultBranchBeforePreflight(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	artifact := planartifact.FromPlans(exactPlan(repo, "release"))

	_, err := ExecuteArtifact(repo, status.Result{State: status.StateBehind}, artifact, git, false, false)
	if err == nil || !strings.Contains(err.Error(), "current default branches are main/main") {
		t.Fatalf("error = %v, want default-branch scope rejection", err)
	}
	assertNoMutation(t, git)
	if len(git.resolveRevisionCalls) != 0 {
		t.Fatalf("resolve calls = %#v, want rejection before stale preflight", git.resolveRevisionCalls)
	}
}

func TestExecuteArtifactRejectsMultipleActionsForCurrentScope(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	planned := exactPlan(repo, "main")
	planned.Actions = append(planned.Actions, planned.Actions[0])

	_, err := ExecuteArtifact(repo, status.Result{State: status.StateBehind}, planartifact.FromPlans(planned), git, false, false)
	if err == nil || !strings.Contains(err.Error(), "at most one default-branch action") {
		t.Fatalf("error = %v, want action cardinality rejection", err)
	}
	assertNoMutation(t, git)
}

func TestExecuteArtifactRejectsNoopWhenCurrentStateDrifted(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	planned := plan.ReconciliationPlan{ID: repo.ID, UID: repo.DurableID(), Actions: []plan.PlannedAction{}}

	_, err := ExecuteArtifact(repo, status.Result{State: status.StateBehind}, planartifact.FromPlans(planned), git, false, true)
	if err == nil || !strings.Contains(err.Error(), "BEHIND requires one non-forced action") {
		t.Fatalf("error = %v, want stale no-op rejection", err)
	}
	assertNoMutation(t, git)
}

func TestExecuteArtifactRejectsActionThatDoesNotMatchCurrentState(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	artifact := planartifact.FromPlans(exactPlan(repo, "main"))

	_, err := ExecuteArtifact(repo, status.Result{State: status.StateDiverged}, artifact, git, true, false)
	if err == nil || !strings.Contains(err.Error(), "DIVERGED requires one forced action") {
		t.Fatalf("error = %v, want current-state policy rejection", err)
	}
	assertNoMutation(t, git)
}

func TestExecuteArtifactRequiresForceAuthorizationAndPreservesAction(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	planned := exactPlan(repo, "main")
	planned.Actions[0].Force = true
	planned.Actions[0].Reason = "mirror is diverged"

	got, err := ExecuteArtifact(repo, status.Result{State: status.StateDiverged}, planartifact.FromPlans(planned), git, false, false)
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("error = %v, want force authorization failure", err)
	}
	if len(got.Actions) != 1 || !got.Actions[0].Force || got.Actions[0].ExpectedOldTarget != testOID {
		t.Fatalf("actions = %#v, want exact forced action preserved", got.Actions)
	}
	assertNoMutation(t, git)
}

func TestExecuteArtifactAllowsForcedDryRunWithoutAuthorization(t *testing.T) {
	git := &fakeGit{}
	repo := testRepo()
	planned := exactPlan(repo, "main")
	planned.Actions[0].Force = true
	planned.Actions[0].Reason = "mirror is ahead"

	got, err := ExecuteArtifact(repo, status.Result{State: status.StateAhead}, planartifact.FromPlans(planned), git, false, true)
	if err != nil {
		t.Fatalf("ExecuteArtifact returned error: %v", err)
	}
	if len(got.Actions) != 1 || !got.Actions[0].Force || !got.DryRun {
		t.Fatalf("result = %#v, want forced dry-run preview", got)
	}
	assertNoMutation(t, git)
}

func TestExecuteArtifactDryRunRejectsStaleRefsWithoutMutation(t *testing.T) {
	git := &staleArtifactGit{}
	repo := testRepo()
	artifact := planartifact.FromPlans(exactPlan(repo, "main"))

	got, err := ExecuteArtifact(repo, status.Result{State: status.StateBehind}, artifact, git, false, true)
	if err == nil || !strings.Contains(err.Error(), "stale action") {
		t.Fatalf("error = %v, want stale preflight failure", err)
	}
	if len(got.Actions) != 1 || !got.DryRun {
		t.Fatalf("result = %#v, want planned dry-run action", got)
	}
	assertNoMutation(t, &git.fakeGit)
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

func exactPlan(repo config.Repo, branch string) plan.ReconciliationPlan {
	return plan.ReconciliationPlan{
		ID:  repo.ID,
		UID: repo.DurableID(),
		Actions: []plan.PlannedAction{{
			Type:              plan.ActionPushBranch,
			Source:            plan.Remote{Provider: "gitlab", Name: "canonical", Branch: branch},
			Target:            plan.Remote{Provider: "github", Name: "mirror", Branch: branch},
			ExpectedSource:    testOID,
			ExpectedOldTarget: testOID,
			Reason:            "mirror is behind",
		}},
	}
}

type staleArtifactGit struct {
	fakeGit
}

func (g *staleArtifactGit) ResolveRevision(repoPath, rev string) (string, error) {
	g.resolveRevisionCalls = append(g.resolveRevisionCalls, rev)
	return "3333333333333333333333333333333333333333", nil
}
