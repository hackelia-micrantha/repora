package apply

import (
	"strings"
	"testing"

	"repoctl/internal/journal"
	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

func TestPreflightRepositoryArtifactAuditedRebindsByTargetIdentity(t *testing.T) {
	repo := multiMirrorRepo()
	artifact := multiPreflightArtifact(t, repo, []plan.PlannedAction{
		multiPreflightAction(repo, 0, "mirror-1", false),
	})
	observed := status.RepositoryResult{
		ID:  repo.ID,
		UID: repo.DurableID(),
		Mirrors: []status.MirrorResult{
			{Target: "gitlab:backup/payments-api", Provider: "gitlab", Path: "backup/payments-api", State: status.StateEqual},
			{Target: "github:org/payments-api", Provider: "github", Path: "org/payments-api", State: status.StateBehind, Behind: 1},
		},
	}
	git := &fakeGit{}
	writer := &recordingJournalWriter{}

	got, err := PreflightRepositoryArtifactAudited(repo, observed, artifact, git, Audit{ExecutionID: "run-multi-dry", Writer: writer})
	if err != nil {
		t.Fatalf("PreflightRepositoryArtifactAudited returned error: %v", err)
	}
	assertNoMutation(t, git)
	if strings.Join(git.resolveRemoteHeadBranchCalls, ",") != "canonical,mirror-0" {
		t.Fatalf("HEAD calls = %#v, want current configured alias", git.resolveRemoteHeadBranchCalls)
	}
	if len(got.Actions) != 1 || got.Actions[0].Target != "github:org/payments-api/main" {
		t.Fatalf("actions = %#v, want stable target identity", got.Actions)
	}
	if len(writer.records) != 2 || writer.records[0].Version != journal.PathVersion || writer.records[1].Actions[0].Outcome != journal.OutcomeValidated {
		t.Fatalf("records = %#v, want path-bound validated evidence", writer.records)
	}
	if writer.records[0].Actions[0].Target.Path != "org/payments-api" {
		t.Fatalf("intent target = %#v, want provider path", writer.records[0].Actions[0].Target)
	}
}

func TestPreflightRepositoryArtifactAuditedRejectsUnknownTargetBeforeGitReads(t *testing.T) {
	repo := multiMirrorRepo()
	action := multiPreflightAction(repo, 0, "mirror-0", false)
	action.Target.Path = "unknown/payments-api"
	artifact := multiPreflightArtifact(t, repo, []plan.PlannedAction{action})
	observed := status.RepositoryResult{
		ID:  repo.ID,
		UID: repo.DurableID(),
		Mirrors: []status.MirrorResult{
			{Target: "github:org/payments-api", State: status.StateBehind},
			{Target: "gitlab:backup/payments-api", State: status.StateEqual},
		},
	}
	git := &fakeGit{}
	writer := &recordingJournalWriter{}

	_, err := PreflightRepositoryArtifactAudited(repo, observed, artifact, git, Audit{ExecutionID: "run-unknown", Writer: writer})
	if err == nil || !strings.Contains(err.Error(), "unknown configured mirror") {
		t.Fatalf("error = %v, want unknown target rejection", err)
	}
	if len(git.resolveRemoteHeadBranchCalls) != 0 || len(git.resolveRevisionCalls) != 0 || len(writer.records) != 0 {
		t.Fatalf("git heads=%#v refs=%#v records=%#v, want rejection before reads or journal", git.resolveRemoteHeadBranchCalls, git.resolveRevisionCalls, writer.records)
	}
}

func TestPreflightRepositoryArtifactAuditedChecksEveryTargetBeforeMutation(t *testing.T) {
	repo := multiMirrorRepo()
	artifact := multiPreflightArtifact(t, repo, []plan.PlannedAction{
		multiPreflightAction(repo, 0, "mirror-0", false),
		multiPreflightAction(repo, 1, "mirror-1", true),
	})
	observed := status.RepositoryResult{
		ID:  repo.ID,
		UID: repo.DurableID(),
		Mirrors: []status.MirrorResult{
			{Target: "github:org/payments-api", State: status.StateBehind},
			{Target: "gitlab:backup/payments-api", State: status.StateDiverged, Ahead: 1, Behind: 1},
		},
	}
	git := &staleSecondMirrorGit{}
	writer := &recordingJournalWriter{}

	_, err := PreflightRepositoryArtifactAudited(repo, observed, artifact, git, Audit{ExecutionID: "run-stale", Writer: writer})
	if err == nil || !strings.Contains(err.Error(), "stale action 1") {
		t.Fatalf("error = %v, want second target stale failure", err)
	}
	assertNoMutation(t, &git.fakeGit)
	if len(writer.records) != 2 || writer.records[1].Actions[0].Outcome != journal.OutcomeSkipped || writer.records[1].Actions[1].Outcome != journal.OutcomeStale {
		t.Fatalf("records = %#v, want skipped then stale result evidence", writer.records)
	}
}

func multiPreflightArtifact(t *testing.T, repo config.Repo, actions []plan.PlannedAction) planartifact.Artifact {
	t.Helper()
	artifact, err := planartifact.FromCurrentPlans(plan.ReconciliationPlan{ID: repo.ID, UID: repo.DurableID(), Actions: actions})
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func multiPreflightAction(repo config.Repo, mirrorIndex int, serializedAlias string, force bool) plan.PlannedAction {
	return plan.PlannedAction{
		Type: plan.ActionPushBranch,
		Source: plan.Remote{
			Provider: repo.Canonical.Provider,
			Path:     repo.Canonical.Path,
			Name:     "canonical",
			Branch:   "main",
		},
		Target: plan.Remote{
			Provider: repo.Mirrors[mirrorIndex].Provider,
			Path:     repo.Mirrors[mirrorIndex].Path,
			Name:     serializedAlias,
			Branch:   "main",
		},
		ExpectedSource:    testOID,
		ExpectedOldTarget: testOID,
		Force:             force,
		Reason:            "mirror requires reconciliation",
	}
}

type staleSecondMirrorGit struct {
	fakeGit
}

func (g *staleSecondMirrorGit) ResolveRevision(repoPath, rev string) (string, error) {
	g.resolveRevisionCalls = append(g.resolveRevisionCalls, rev)
	if rev == "refs/remotes/mirror-1/main" {
		return strings.Repeat("b", 40), nil
	}
	return testOID, nil
}
