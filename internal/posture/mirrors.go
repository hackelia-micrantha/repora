package posture

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/status"
)

const (
	MirrorInventoryKind    = "repora.posture-mirrors"
	MirrorInventoryVersion = 1
)

type MirrorEndpointIdentity struct {
	Provider string `json:"provider"`
	Path     string `json:"path"`
}

type MirrorCanonicalFacts struct {
	Identity                   MirrorEndpointIdentity `json:"identity"`
	DefaultBranch              Fact[string]           `json:"default_branch"`
	Commit                     Fact[string]           `json:"commit"`
	Visibility                 Fact[string]           `json:"visibility"`
	CurrentActorPushPermission Fact[bool]             `json:"current_actor_push_permission"`
}

type MirrorTargetFacts struct {
	Identity                   MirrorEndpointIdentity `json:"identity"`
	CacheRemote                Fact[string]           `json:"cache_remote"`
	DefaultBranch              Fact[string]           `json:"default_branch"`
	DefaultBranchDrift         Fact[bool]             `json:"default_branch_drift"`
	Commit                     Fact[string]           `json:"commit"`
	Divergence                 Fact[string]           `json:"divergence"`
	Ahead                      Fact[int]              `json:"ahead"`
	Behind                     Fact[int]              `json:"behind"`
	Visibility                 Fact[string]           `json:"visibility"`
	CurrentActorPushPermission Fact[bool]             `json:"current_actor_push_permission"`
	TagDrift                   Fact[bool]             `json:"tag_drift"`
	ReleaseDrift               Fact[bool]             `json:"release_drift"`
}

type MirrorRepositoryFacts struct {
	ID        string               `json:"id"`
	UID       string               `json:"uid"`
	Mode      Fact[string]         `json:"mode"`
	Direction Fact[string]         `json:"direction"`
	Canonical MirrorCanonicalFacts `json:"canonical"`
	Mirrors   []MirrorTargetFacts  `json:"mirrors"`
	Evidence  []Evidence           `json:"evidence"`
}

type MirrorInventory struct {
	Kind     string                  `json:"kind"`
	Version  int                     `json:"version"`
	Repos    []MirrorRepositoryFacts `json:"repos"`
	Evidence []Evidence              `json:"evidence"`
}

type MirrorBranchObservation struct {
	Name      string
	Available bool
	Evidence  Evidence
}

type MirrorLocalObservation struct {
	Status          status.RepositoryResult
	CanonicalBranch MirrorBranchObservation
	MirrorBranches  []MirrorBranchObservation
}

type MirrorLocalObserver interface {
	Observe(config.Repo) (MirrorLocalObservation, error)
}

type GitMirrorLocalObserver struct{}

func (GitMirrorLocalObserver) Observe(repo config.Repo) (MirrorLocalObservation, error) {
	client := gitwrap.Client{}
	result, err := status.CheckAll(repo, client)
	if err != nil && result.Error != "" {
		return MirrorLocalObservation{}, err
	}
	cachePath, err := gitwrap.MirrorPath(repo.DurableID())
	if err != nil {
		return MirrorLocalObservation{}, err
	}
	out := MirrorLocalObservation{Status: result, MirrorBranches: make([]MirrorBranchObservation, len(repo.Mirrors))}
	out.CanonicalBranch = observeRemoteHead(client, cachePath, repo.DurableID(), "canonical")
	for i := range repo.Mirrors {
		out.MirrorBranches[i] = observeRemoteHead(client, cachePath, repo.DurableID(), "mirror-"+strconv.Itoa(i))
	}
	return out, nil
}

func observeRemoteHead(client gitwrap.Client, cachePath, uid, remote string) MirrorBranchObservation {
	branch, err := client.ResolveRemoteHeadBranch(cachePath, remote)
	evidence := Evidence{Source: "git.remote_head", Reference: uid + ":" + remote + "/HEAD"}
	if err != nil || strings.TrimSpace(branch) == "" {
		evidence.Detail = "remote HEAD unavailable"
		return MirrorBranchObservation{Evidence: evidence}
	}
	return MirrorBranchObservation{Name: strings.TrimSpace(branch), Available: true, Evidence: evidence}
}

type MirrorProviderRepository struct {
	DefaultBranch  string
	Visibility     string
	PushPermission *bool
}

type MirrorProviderReader interface {
	Repository(context.Context, config.Endpoint) (MirrorProviderRepository, ReadObservation, error)
}

// DefaultMirrorProviderReader reuses the read-only GitHub repository reader.
// The current shared reader exposes default-branch identity only; visibility and
// current-actor push permission therefore remain unknown for GitHub and
// unavailable for providers without a posture adapter.
type DefaultMirrorProviderReader struct {
	GitHub GitHubReader
}

func (r DefaultMirrorProviderReader) Repository(ctx context.Context, endpoint config.Endpoint) (MirrorProviderRepository, ReadObservation, error) {
	identity, err := mirrorEndpointIdentity(endpoint)
	if err != nil {
		return MirrorProviderRepository{}, ReadObservation{}, err
	}
	if identity.Provider != "github" {
		return MirrorProviderRepository{}, ReadObservation{Available: false, Evidence: Evidence{
			Source: "provider.unsupported", Reference: identity.Provider + ":" + identity.Path,
			Detail: "provider metadata adapter is not implemented for mirror posture v1",
		}}, nil
	}
	if r.GitHub == nil {
		return MirrorProviderRepository{}, ReadObservation{}, fmt.Errorf("GitHub mirror provider reader is required")
	}
	repository, obs, err := r.GitHub.Repository(ctx, identity.Path)
	if err != nil || !obs.Available {
		return MirrorProviderRepository{}, obs, err
	}
	return MirrorProviderRepository{DefaultBranch: repository.DefaultBranch}, obs, nil
}

func NewMirrorInventory() MirrorInventory {
	return MirrorInventory{Kind: MirrorInventoryKind, Version: MirrorInventoryVersion, Repos: []MirrorRepositoryFacts{}, Evidence: []Evidence{}}
}

func CollectMirrorPosture(ctx context.Context, spec config.Spec, local MirrorLocalObserver, providers MirrorProviderReader) (MirrorInventory, error) {
	if local == nil || providers == nil {
		return MirrorInventory{}, fmt.Errorf("mirror local observer and provider reader are required")
	}
	inventory := NewMirrorInventory()
	for _, repo := range spec.Repos {
		facts, err := collectMirrorRepository(ctx, repo, local, providers)
		if err != nil {
			return MirrorInventory{}, fmt.Errorf("collect mirror posture for repo %q: %w", repo.ID, err)
		}
		inventory.Repos = append(inventory.Repos, facts)
	}
	return inventory, inventory.Validate()
}

func collectMirrorRepository(ctx context.Context, repo config.Repo, local MirrorLocalObserver, providers MirrorProviderReader) (MirrorRepositoryFacts, error) {
	canonicalIdentity, err := mirrorEndpointIdentity(repo.Canonical)
	if err != nil {
		return MirrorRepositoryFacts{}, fmt.Errorf("canonical identity: %w", err)
	}
	observed, err := local.Observe(repo)
	if err != nil {
		return MirrorRepositoryFacts{}, fmt.Errorf("observe reconciliation state: %w", err)
	}
	if len(observed.Status.Mirrors) != len(repo.Mirrors) || len(observed.MirrorBranches) != len(repo.Mirrors) {
		return MirrorRepositoryFacts{}, fmt.Errorf("mirror observation count does not match declared topology")
	}
	configEvidence := Evidence{Source: "repora.config", Reference: repo.DurableID()}
	facts := MirrorRepositoryFacts{
		ID: repo.ID, UID: repo.DurableID(),
		Mode: Observed(repo.Mode, configEvidence),
		Direction: Observed("canonical_to_mirror", Evidence{Source: "repora.mirror_semantics", Reference: repo.DurableID()}),
		Canonical: MirrorCanonicalFacts{Identity: canonicalIdentity}, Mirrors: []MirrorTargetFacts{}, Evidence: []Evidence{configEvidence},
	}
	facts.Canonical.DefaultBranch = branchFact(observed.CanonicalBranch)
	facts.Canonical.Commit = commitFact(observed.Status.Canonical.Commit, Evidence{Source: "git.reconciliation", Reference: repo.DurableID() + ":canonical/HEAD"})
	canonicalProvider, canonicalObs, err := providers.Repository(ctx, repo.Canonical)
	if err != nil {
		return MirrorRepositoryFacts{}, fmt.Errorf("read canonical provider metadata: %w", err)
	}
	facts.Canonical.Visibility = providerVisibilityFact(canonicalProvider, canonicalObs)
	facts.Canonical.CurrentActorPushPermission = providerPushFact(canonicalProvider, canonicalObs)
	if facts.Canonical.DefaultBranch.State != StateObserved && canonicalObs.Available && canonicalProvider.DefaultBranch != "" {
		facts.Canonical.DefaultBranch = Observed(canonicalProvider.DefaultBranch, canonicalObs.Evidence)
	}

	for i, endpoint := range repo.Mirrors {
		identity, err := mirrorEndpointIdentity(endpoint)
		if err != nil {
			return MirrorRepositoryFacts{}, fmt.Errorf("mirror[%d] identity: %w", i, err)
		}
		statusResult := observed.Status.Mirrors[i]
		providerRepo, providerObs, err := providers.Repository(ctx, endpoint)
		if err != nil {
			return MirrorRepositoryFacts{}, fmt.Errorf("read mirror[%d] provider metadata: %w", i, err)
		}
		defaultBranch := branchFact(observed.MirrorBranches[i])
		if defaultBranch.State != StateObserved && providerObs.Available && providerRepo.DefaultBranch != "" {
			defaultBranch = Observed(providerRepo.DefaultBranch, providerObs.Evidence)
		}
		mirror := MirrorTargetFacts{
			Identity: identity,
			CacheRemote: Observed("mirror-"+strconv.Itoa(i), Evidence{Source: "repora.cache_remote", Reference: repo.DurableID()}),
			DefaultBranch: defaultBranch,
			Commit: commitFact(statusResult.Commit, Evidence{Source: "git.reconciliation", Reference: identity.Provider + ":" + identity.Path + ":HEAD"}),
			Divergence: divergenceFact(statusResult), Ahead: countFact(statusResult, true), Behind: countFact(statusResult, false),
			Visibility: providerVisibilityFact(providerRepo, providerObs),
			CurrentActorPushPermission: providerPushFact(providerRepo, providerObs),
			TagDrift: Unknown[bool](Evidence{Source: "repora.ref_scope", Reference: identity.Provider + ":" + identity.Path, Detail: "tag drift is outside mirror posture v1 default-branch scope"}),
			ReleaseDrift: Unknown[bool](Evidence{Source: "repora.ref_scope", Reference: identity.Provider + ":" + identity.Path, Detail: "release drift requires provider release adapters not implemented in v1"}),
		}
		mirror.DefaultBranchDrift = branchDriftFact(facts.Canonical.DefaultBranch, mirror.DefaultBranch)
		facts.Mirrors = append(facts.Mirrors, mirror)
	}
	return facts, nil
}

func branchFact(observation MirrorBranchObservation) Fact[string] {
	if observation.Available && strings.TrimSpace(observation.Name) != "" {
		return Observed(strings.TrimSpace(observation.Name), observation.Evidence)
	}
	return Unavailable[string](observation.Evidence)
}

func commitFact(commit string, evidence Evidence) Fact[string] {
	if strings.TrimSpace(commit) == "" {
		return Unavailable[string](evidenceWithDetail(evidence, "default-branch commit unavailable"))
	}
	return Observed(strings.TrimSpace(commit), evidence)
}

func divergenceFact(result status.MirrorResult) Fact[string] {
	evidence := Evidence{Source: "git.reconciliation", Reference: result.Target + ":HEAD"}
	if result.State == status.StateError {
		return Unavailable[string](evidenceWithDetail(evidence, "mirror reconciliation observation unavailable"))
	}
	return Observed(string(result.State), evidence)
}

func countFact(result status.MirrorResult, ahead bool) Fact[int] {
	evidence := Evidence{Source: "git.reconciliation", Reference: result.Target + ":HEAD"}
	if result.State == status.StateError {
		return Unavailable[int](evidenceWithDetail(evidence, "mirror reconciliation count unavailable"))
	}
	if ahead { return Observed(result.Ahead, evidence) }
	return Observed(result.Behind, evidence)
}

func branchDriftFact(canonical, mirror Fact[string]) Fact[bool] {
	evidence := append(cloneEvidence(canonical.Evidence), mirror.Evidence...)
	if canonical.State != StateObserved || canonical.Value == nil || mirror.State != StateObserved || mirror.Value == nil {
		if canonical.State == StateUnavailable || mirror.State == StateUnavailable { return Unavailable[bool](evidence...) }
		return Unknown[bool](evidence...)
	}
	return Observed(*canonical.Value != *mirror.Value, evidence...)
}

func providerVisibilityFact(repository MirrorProviderRepository, observation ReadObservation) Fact[string] {
	if !observation.Available { return Unavailable[string](observation.Evidence) }
	if strings.TrimSpace(repository.Visibility) == "" { return Unknown[string](observation.Evidence) }
	return Observed(strings.TrimSpace(repository.Visibility), observation.Evidence)
}

func providerPushFact(repository MirrorProviderRepository, observation ReadObservation) Fact[bool] {
	if !observation.Available { return Unavailable[bool](observation.Evidence) }
	if repository.PushPermission == nil { return Unknown[bool](observation.Evidence) }
	return Observed(*repository.PushPermission, observation.Evidence)
}

func mirrorEndpointIdentity(endpoint config.Endpoint) (MirrorEndpointIdentity, error) {
	provider := strings.TrimSpace(endpoint.Provider)
	pathValue := strings.Trim(strings.TrimSpace(endpoint.Path), "/")
	if pathValue == "" {
		legacy, err := mirrorLegacyPath(endpoint.URL)
		if err != nil { return MirrorEndpointIdentity{}, err }
		pathValue = legacy
	}
	if provider == "" || pathValue == "" { return MirrorEndpointIdentity{}, fmt.Errorf("provider and path are required") }
	return MirrorEndpointIdentity{Provider: provider, Path: pathValue}, nil
}

func mirrorLegacyPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	var pathValue string
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw); if err != nil { return "", fmt.Errorf("parse legacy URL") }
		pathValue = parsed.Path
	} else if at := strings.LastIndex(raw, "@"); at >= 0 {
		if colon := strings.Index(raw[at+1:], ":"); colon >= 0 { pathValue = raw[at+1+colon+1:] }
	}
	pathValue = strings.TrimSuffix(strings.Trim(strings.TrimSpace(pathValue), "/"), ".git")
	parts := strings.Split(pathValue, "/")
	if len(parts) < 2 { return "", fmt.Errorf("legacy URL does not contain a safe repository path") }
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `\\:@?#`) { return "", fmt.Errorf("legacy URL contains an unsafe repository path") }
	}
	return pathValue, nil
}

func (i MirrorInventory) Validate() error {
	if i.Kind != MirrorInventoryKind || i.Version != MirrorInventoryVersion { return fmt.Errorf("unsupported mirror posture contract: kind=%q version=%d", i.Kind, i.Version) }
	if i.Repos == nil || i.Evidence == nil { return fmt.Errorf("mirror posture repos and evidence arrays are required") }
	for ri, repo := range i.Repos {
		if strings.TrimSpace(repo.ID) == "" || strings.TrimSpace(repo.UID) == "" { return fmt.Errorf("repo[%d] id and uid are required", ri) }
		if repo.Mirrors == nil || repo.Evidence == nil { return fmt.Errorf("repo[%d] mirror and evidence arrays are required", ri) }
		if err := validateFact("mode", repo.Mode); err != nil { return err }
		if err := validateFact("direction", repo.Direction); err != nil { return err }
		if err := validateMirrorEndpointFacts("canonical", repo.Canonical.DefaultBranch, repo.Canonical.Commit, repo.Canonical.Visibility, repo.Canonical.CurrentActorPushPermission); err != nil { return err }
		for mi, mirror := range repo.Mirrors {
			if mirror.Identity.Provider == "" || mirror.Identity.Path == "" { return fmt.Errorf("repo[%d] mirror[%d] identity is required", ri, mi) }
			checks := []error{
				validateFact("cache_remote", mirror.CacheRemote), validateFact("default_branch", mirror.DefaultBranch), validateFact("default_branch_drift", mirror.DefaultBranchDrift),
				validateFact("commit", mirror.Commit), validateFact("divergence", mirror.Divergence), validateFact("ahead", mirror.Ahead), validateFact("behind", mirror.Behind),
				validateFact("visibility", mirror.Visibility), validateFact("current_actor_push_permission", mirror.CurrentActorPushPermission), validateFact("tag_drift", mirror.TagDrift), validateFact("release_drift", mirror.ReleaseDrift),
			}
			for _, err := range checks { if err != nil { return fmt.Errorf("repo[%d] mirror[%d]: %w", ri, mi, err) } }
		}
	}
	return nil
}

func validateMirrorEndpointFacts(prefix string, branch, commit, visibility Fact[string], push Fact[bool]) error {
	for _, err := range []error{validateFact("default_branch", branch), validateFact("commit", commit), validateFact("visibility", visibility), validateFact("current_actor_push_permission", push)} {
		if err != nil { return fmt.Errorf("%s: %w", prefix, err) }
	}
	return nil
}

func (i MirrorInventory) Marshal() ([]byte, error) {
	if err := i.Validate(); err != nil { return nil, err }
	data, err := json.MarshalIndent(i, "", "  "); if err != nil { return nil, fmt.Errorf("encode mirror posture inventory: %w", err) }
	return append(data, '\n'), nil
}
