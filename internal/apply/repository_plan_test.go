package apply

import (
	"strings"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

func TestBuildRepositoryArtifactPlansMirrorsInConfigurationOrder(t *testing.T) {
	repo := bitbucketMultiMirrorRepo()
	observed := status.RepositoryResult{
		ID:  repo.ID,
		UID: repo.DurableID(),
		Mirrors: []status.MirrorResult{
			{Target: "bitbucket:workspace/payments-api", Provider: "bitbucket", Path: "workspace/payments-api", State: status.StateBehind, Behind: 2},
			{Target: "github:org/payments-api", Provider: "github", Path: "org/payments-api", State: status.StateEqual},
		},
	}
	git := &fakeGit{}

	artifact, err := BuildRepositoryArtifact(repo, observed, git)
	if err != nil {
		t.Fatalf("BuildRepositoryArtifact returned error: %v", err)
	}
	if artifact.Version != planartifact.Version || len(artifact.Repositories) != 1 || len(artifact.Repositories[0].Actions) != 1 {
		t.Fatalf("artifact = %#v, want one v2 action", artifact)
	}
	action := artifact.Repositories[0].Actions[0]
	if action.Target.Provider != "bitbucket" || action.Target.Path != "workspace/payments-api" || action.Target.Remote != "mirror-1" {
		t.Fatalf("target = %#v, want configured second mirror", action.Target)
	}
	if strings.Join(git.resolveRemoteHeadBranchCalls, ",") != "canonical,mirror-1" {
		t.Fatalf("HEAD calls = %#v, want canonical then required mirror", git.resolveRemoteHeadBranchCalls)
	}
}

func TestBuildRepositoryArtifactUsesTargetIdentityNotObservationOrder(t *testing.T) {
	repo := bitbucketMultiMirrorRepo()
	repo.Mirrors[0], repo.Mirrors[1] = repo.Mirrors[1], repo.Mirrors[0]
	observed := status.RepositoryResult{
		ID:  repo.ID,
		UID: repo.DurableID(),
		Mirrors: []status.MirrorResult{
			{Target: "github:org/payments-api", Provider: "github", Path: "org/payments-api", State: status.StateBehind, Behind: 1},
			{Target: "bitbucket:workspace/payments-api", Provider: "bitbucket", Path: "workspace/payments-api", State: status.StateEqual},
		},
	}

	artifact, err := BuildRepositoryArtifact(repo, observed, &fakeGit{})
	if err != nil {
		t.Fatalf("BuildRepositoryArtifact returned error: %v", err)
	}
	action := artifact.Repositories[0].Actions[0]
	if action.Target.Path != "org/payments-api" || action.Target.Remote != "mirror-1" {
		t.Fatalf("target = %#v, want identity-matched github target at current index 1", action.Target)
	}
}

func TestBuildRepositoryArtifactRecordsForcedIntentPerMirror(t *testing.T) {
	repo := bitbucketMultiMirrorRepo()
	observed := status.RepositoryResult{
		ID:  repo.ID,
		UID: repo.DurableID(),
		Mirrors: []status.MirrorResult{
			{Target: "github:org/payments-api", Provider: "github", Path: "org/payments-api", State: status.StateAhead, Ahead: 1},
			{Target: "bitbucket:workspace/payments-api", Provider: "bitbucket", Path: "workspace/payments-api", State: status.StateDiverged, Ahead: 1, Behind: 2},
		},
	}

	artifact, err := BuildRepositoryArtifact(repo, observed, &fakeGit{})
	if err != nil {
		t.Fatalf("BuildRepositoryArtifact returned error: %v", err)
	}
	if len(artifact.Repositories[0].Actions) != 2 || !artifact.Repositories[0].Actions[0].Force || !artifact.Repositories[0].Actions[1].Force {
		t.Fatalf("actions = %#v, want two forced intents", artifact.Repositories[0].Actions)
	}
}

func TestBuildRepositoryArtifactRejectsIncompleteMirrorStatusBeforeRefReads(t *testing.T) {
	repo := bitbucketMultiMirrorRepo()
	observed := status.RepositoryResult{
		ID:  repo.ID,
		UID: repo.DurableID(),
		Mirrors: []status.MirrorResult{
			{Target: "github:org/payments-api", Provider: "github", Path: "org/payments-api", State: status.StateError, Error: "fetch mirror: unavailable"},
			{Target: "bitbucket:workspace/payments-api", Provider: "bitbucket", Path: "workspace/payments-api", State: status.StateEqual},
		},
	}
	git := &fakeGit{}

	_, err := BuildRepositoryArtifact(repo, observed, git)
	if err == nil || !strings.Contains(err.Error(), "status is incomplete") {
		t.Fatalf("error = %v, want incomplete status rejection", err)
	}
	if len(git.resolveRemoteHeadBranchCalls) != 1 || len(git.resolveRevisionCalls) != 0 {
		t.Fatalf("git calls = heads %#v refs %#v, want no ref reads after status rejection", git.resolveRemoteHeadBranchCalls, git.resolveRevisionCalls)
	}
}

func bitbucketMultiMirrorRepo() config.Repo {
	return config.Repo{
		ID:        "payments-api",
		UID:       "repo.org.payments-api",
		Canonical: config.Endpoint{Provider: "gitlab", Path: "org/payments-api"},
		Mirrors: []config.Endpoint{
			{Provider: "github", Path: "org/payments-api"},
			{Provider: "bitbucket", Path: "workspace/payments-api"},
		},
		Mode: "mirror",
	}
}

func multiMirrorRepo() config.Repo {
	return config.Repo{
		ID:        "payments-api",
		UID:       "repo.org.payments-api",
		Canonical: config.Endpoint{Provider: "gitlab", Path: "org/payments-api"},
		Mirrors: []config.Endpoint{
			{Provider: "github", Path: "org/payments-api"},
			{Provider: "gitlab", Path: "backup/payments-api"},
		},
		Mode: "mirror",
	}
}
