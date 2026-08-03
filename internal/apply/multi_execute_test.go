package apply

import (
	"errors"
	"strings"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/journal"
	"repoctl/internal/plan"
	"repoctl/internal/status"
)

func TestExecuteRepositoryArtifactAuditedPreservesPartialOutcomes(t *testing.T) {
	repo := multiMirrorRepo()
	repo.Mirrors = append(repo.Mirrors, config.Endpoint{Provider: "github", Path: "third/payments-api"})
	artifact := multiPreflightArtifact(t, repo, []plan.PlannedAction{
		multiPreflightAction(repo, 0, "serialized-zero", false),
		multiPreflightAction(repo, 1, "serialized-one", false),
		multiPreflightAction(repo, 2, "serialized-two", false),
	})
	observed := status.RepositoryResult{
		ID: repo.ID, UID: repo.DurableID(),
		Mirrors: []status.MirrorResult{
			{Target: "github:org/payments-api", State: status.StateBehind},
			{Target: "gitlab:backup/payments-api", State: status.StateBehind},
			{Target: "github:third/payments-api", State: status.StateBehind},
		},
	}
	git := &partialExecutionGit{failRemote: "mirror-1"}
	writer := &recordingJournalWriter{}

	got, err := ExecuteRepositoryArtifactAudited(repo, observed, artifact, git, false, false, Audit{ExecutionID: "run-partial", Writer: writer})
	if err == nil || !strings.Contains(err.Error(), "mirror-1") {
		t.Fatalf("error = %v, want middle mirror runtime failure", err)
	}
	if len(git.pushBranchCalls) != 3 {
		t.Fatalf("push calls = %#v, want all mirrors attempted", git.pushBranchCalls)
	}
	if len(got.Actions) != 3 || got.Actions[0].Outcome != string(journal.OutcomeApplied) || got.Actions[1].Outcome != string(journal.OutcomeFailed) || got.Actions[2].Outcome != string(journal.OutcomeApplied) {
		t.Fatalf("actions = %#v, want applied/failed/applied", got.Actions)
	}
	if got.Actions[0].After != testOID || got.Actions[2].After != testOID || got.Applied {
		t.Fatalf("result = %#v, want partial non-atomic outcome", got)
	}
	if len(writer.records) != 2 {
		t.Fatalf("records = %#v, want intent/result pair", writer.records)
	}
	final := writer.records[1]
	if final.Actions[0].Outcome != journal.OutcomeApplied || final.Actions[1].Outcome != journal.OutcomeFailed || final.Actions[2].Outcome != journal.OutcomeApplied {
		t.Fatalf("journal outcomes = %#v", final.Actions)
	}
	if final.Actions[0].Target.Path != "org/payments-api" || final.Actions[2].Target.Path != "third/payments-api" {
		t.Fatalf("journal targets = %#v", final.Actions)
	}
}

func TestExecuteRepositoryArtifactAuditedRedactsEmbeddedLocalPath(t *testing.T) {
	repo := multiMirrorRepo()
	artifact := multiPreflightArtifact(t, repo, []plan.PlannedAction{
		multiPreflightAction(repo, 0, "mirror-0", false),
	})
	observed := status.RepositoryResult{
		ID: repo.ID, UID: repo.DurableID(),
		Mirrors: []status.MirrorResult{
			{Target: "github:org/payments-api", State: status.StateBehind},
			{Target: "gitlab:backup/payments-api", State: status.StateEqual},
		},
	}
	git := &partialExecutionGit{
		failRemote: "mirror-0",
		failErr:    errors.New("fatal: '/tmp/private/mirror.git' does not appear to be a git repository"),
	}
	writer := &recordingJournalWriter{}

	got, err := ExecuteRepositoryArtifactAudited(repo, observed, artifact, git, false, false, Audit{ExecutionID: "run-redacted", Writer: writer})
	if err == nil || strings.Contains(err.Error(), "/tmp/private") {
		t.Fatalf("error = %v, want sanitized public failure", err)
	}
	if got.Error != "execution diagnostic redacted" || got.Actions[0].Error != "execution diagnostic redacted" {
		t.Fatalf("result = %#v, want redacted aggregate and action diagnostics", got)
	}
	if len(writer.records) != 2 || writer.records[1].Actions[0].Error != "execution diagnostic redacted" {
		t.Fatalf("records = %#v, want redacted durable diagnostic", writer.records)
	}
}

func TestExecuteRepositoryArtifactAuditedRequiresForceBeforeIntent(t *testing.T) {
	repo := multiMirrorRepo()
	artifact := multiPreflightArtifact(t, repo, []plan.PlannedAction{
		multiPreflightAction(repo, 0, "mirror-0", true),
	})
	observed := status.RepositoryResult{
		ID: repo.ID, UID: repo.DurableID(),
		Mirrors: []status.MirrorResult{
			{Target: "github:org/payments-api", State: status.StateDiverged},
			{Target: "gitlab:backup/payments-api", State: status.StateEqual},
		},
	}
	git := &partialExecutionGit{}
	writer := &recordingJournalWriter{}

	_, err := ExecuteRepositoryArtifactAudited(repo, observed, artifact, git, false, false, Audit{ExecutionID: "run-force", Writer: writer})
	if !errors.Is(err, ErrForceAuthorization) {
		t.Fatalf("error = %v, want force authorization", err)
	}
	if len(writer.records) != 0 || len(git.pushBranchCalls) != 0 || len(git.resolveRemoteHeadBranchCalls) != 0 {
		t.Fatalf("records=%#v heads=%#v pushes=%#v, want pre-intent refusal", writer.records, git.resolveRemoteHeadBranchCalls, git.pushBranchCalls)
	}
}

type partialExecutionGit struct {
	fakeGit
	failRemote string
	failErr    error
}

func (g *partialExecutionGit) PushBranch(repoPath, remote, srcRef, dstBranch string) error {
	g.fakeGit.PushBranch(repoPath, remote, srcRef, dstBranch)
	if remote == g.failRemote {
		if g.failErr != nil {
			return g.failErr
		}
		return errors.New(remote + " unavailable")
	}
	return nil
}
