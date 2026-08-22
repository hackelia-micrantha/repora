package posture

import (
	"context"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/status"
)

type fakeMirrorLocalObserver struct {
	observation MirrorLocalObservation
	err         error
}

func (f fakeMirrorLocalObserver) Observe(config.Repo) (MirrorLocalObservation, error) {
	return f.observation, f.err
}

type fakeMirrorProviderReader struct {
	values map[string]MirrorProviderRepository
	obs    map[string]ReadObservation
}

func (f fakeMirrorProviderReader) Repository(_ context.Context, endpoint config.Endpoint) (MirrorProviderRepository, ReadObservation, error) {
	identity, err := mirrorEndpointIdentity(endpoint)
	if err != nil {
		return MirrorProviderRepository{}, ReadObservation{}, err
	}
	key := identity.Provider + ":" + identity.Path
	return f.values[key], f.obs[key], nil
}

func mirrorTestRepo() config.Repo {
	return config.Repo{
		ID: "example", UID: "repo.example", Mode: "mirror",
		Canonical: config.Endpoint{Provider: "gitlab", Path: "acme/example"},
		Mirrors:   []config.Endpoint{{Provider: "github", Path: "acme/example"}},
	}
}

func mirrorLocal(branch string, mirrorState status.State) MirrorLocalObservation {
	return MirrorLocalObservation{
		Status: status.RepositoryResult{
			ID: "example", UID: "repo.example",
			Canonical: status.RefResult{Ref: "HEAD", Commit: "abc1234"},
			Mirrors:   []status.MirrorResult{{Target: "github:acme/example", Provider: "github", Path: "acme/example", Ref: "HEAD", Commit: "abc1234", State: mirrorState}},
		},
		CanonicalBranch: MirrorBranchObservation{Name: "main", Available: true, Evidence: Evidence{Source: "git.remote_head", Reference: "canonical"}},
		MirrorBranches:  []MirrorBranchObservation{{Name: branch, Available: true, Evidence: Evidence{Source: "git.remote_head", Reference: "mirror-0"}}},
	}
}

func providerFacts() fakeMirrorProviderReader {
	push := false
	return fakeMirrorProviderReader{
		values: map[string]MirrorProviderRepository{
			"gitlab:acme/example": {DefaultBranch: "main", Visibility: "private", PushPermission: &push},
			"github:acme/example": {DefaultBranch: "main", Visibility: "public", PushPermission: &push},
		},
		obs: map[string]ReadObservation{
			"gitlab:acme/example": {Available: true, Evidence: Evidence{Source: "fake.provider", Reference: "gitlab:acme/example"}},
			"github:acme/example": {Available: true, Evidence: Evidence{Source: "fake.provider", Reference: "github:acme/example"}},
		},
	}
}

func TestCollectMirrorPostureInSync(t *testing.T) {
	repo := mirrorTestRepo()
	inventory, err := CollectMirrorPosture(context.Background(), config.Spec{Repos: []config.Repo{repo}}, fakeMirrorLocalObserver{observation: mirrorLocal("main", status.StateEqual)}, providerFacts())
	if err != nil {
		t.Fatal(err)
	}
	mirror := inventory.Repos[0].Mirrors[0]
	if mirror.DefaultBranchDrift.Value == nil || *mirror.DefaultBranchDrift.Value {
		t.Fatalf("default branch drift = %#v", mirror.DefaultBranchDrift)
	}
	if mirror.Divergence.Value == nil || *mirror.Divergence.Value != "EQUAL" {
		t.Fatalf("divergence = %#v", mirror.Divergence)
	}
	if mirror.Visibility.Value == nil || *mirror.Visibility.Value != "public" {
		t.Fatalf("visibility = %#v", mirror.Visibility)
	}
	if mirror.TagDrift.State != StateUnknown || mirror.ReleaseDrift.State != StateUnknown {
		t.Fatalf("tag/release scope = %#v %#v", mirror.TagDrift, mirror.ReleaseDrift)
	}
}

func TestCollectMirrorPostureDetectsDefaultBranchDrift(t *testing.T) {
	repo := mirrorTestRepo()
	inventory, err := CollectMirrorPosture(context.Background(), config.Spec{Repos: []config.Repo{repo}}, fakeMirrorLocalObserver{observation: mirrorLocal("master", status.StateEqual)}, providerFacts())
	if err != nil {
		t.Fatal(err)
	}
	fact := inventory.Repos[0].Mirrors[0].DefaultBranchDrift
	if fact.State != StateObserved || fact.Value == nil || !*fact.Value {
		t.Fatalf("default branch drift = %#v", fact)
	}
}

func TestCollectMirrorPosturePreservesUnavailableProviderEvidence(t *testing.T) {
	repo := mirrorTestRepo()
	providers := providerFacts()
	providers.obs["github:acme/example"] = ReadObservation{Available: false, Evidence: Evidence{Source: "github.repository", Reference: "acme/example", Detail: "HTTP 403"}}
	inventory, err := CollectMirrorPosture(context.Background(), config.Spec{Repos: []config.Repo{repo}}, fakeMirrorLocalObserver{observation: mirrorLocal("main", status.StateEqual)}, providers)
	if err != nil {
		t.Fatal(err)
	}
	mirror := inventory.Repos[0].Mirrors[0]
	if mirror.Visibility.State != StateUnavailable || mirror.CurrentActorPushPermission.State != StateUnavailable {
		t.Fatalf("provider facts = %#v %#v", mirror.Visibility, mirror.CurrentActorPushPermission)
	}
	if mirror.DefaultBranchDrift.State != StateObserved || mirror.DefaultBranchDrift.Value == nil || *mirror.DefaultBranchDrift.Value {
		t.Fatalf("local branch evidence should remain authoritative: %#v", mirror.DefaultBranchDrift)
	}
}

func TestCollectMirrorPostureDoesNotInventDriftWhenMirrorObservationFails(t *testing.T) {
	repo := mirrorTestRepo()
	local := mirrorLocal("", status.StateError)
	local.MirrorBranches[0] = MirrorBranchObservation{Evidence: Evidence{Source: "git.remote_head", Reference: "mirror-0", Detail: "unavailable"}}
	local.Status.Mirrors[0].Commit = ""
	providers := providerFacts()
	providers.obs["github:acme/example"] = ReadObservation{Available: false, Evidence: Evidence{Source: "github.repository", Reference: "acme/example"}}
	inventory, err := CollectMirrorPosture(context.Background(), config.Spec{Repos: []config.Repo{repo}}, fakeMirrorLocalObserver{observation: local}, providers)
	if err != nil {
		t.Fatal(err)
	}
	mirror := inventory.Repos[0].Mirrors[0]
	if mirror.DefaultBranchDrift.State != StateUnavailable || mirror.Divergence.State != StateUnavailable || mirror.Commit.State != StateUnavailable {
		t.Fatalf("failed mirror facts = %#v", mirror)
	}
}
