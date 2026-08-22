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

	cachePath, pathErr := gitwrap.MirrorPath(repo.DurableID())
	if pathErr != nil {
		return MirrorLocalObservation{}, pathErr
	}
	observation := MirrorLocalObservation{
		Status:         result,
		MirrorBranches: make([]MirrorBranchObservation, len(repo.Mirrors)),
	}
	canonicalBranch, branchErr := client.ResolveRemoteHeadBranch(cachePath, "canonical")
	if branchErr == nil {
		observation.CanonicalBranch = MirrorBranchObservation{
			Name:      strings.TrimSpace(canonicalBranch),
			Available: true,
			Evidence:  Evidence{Source: "git.remote_head", Reference: repo.DurableID() + ":canonical/HEAD"},
		}
	} else {
		observation.CanonicalBranch = MirrorBranchObservation{
			Evidence: Evidence{Source: "git.remote_head", Reference: repo.DurableID() + ":canonical/HEAD", Detail: "canonical remote HEAD unavailable"},
		}
	}
	for i := range repo.Mirrors {
		remote := "mirror-" + strconv.Itoa(i)
		branch, branchErr := client.ResolveRemoteHeadBranch(cachePath, remote)
		if branchErr == nil {
			observation.MirrorBranches[i] = MirrorBranchObservation{
				Name:      strings.TrimSpace(branch),
				Available: true,
				Evidence:  Evidence{Source: "git.remote_head", Reference: repo.DurableID() + ":" + remote + "/HEAD"},
			}
		} else {
			observation.MirrorBranches[i] = MirrorBranchObservation{
				Evidence: Evidence{Source: "git.remote_head", Reference: repo.DurableID() + ":" + remote + "/HEAD", Detail: "mirror remote HEAD unavailable"},
			}
		}
	}
	return observation, nil
}

type MirrorProviderRepository struct {
	DefaultBranch  string
	Visibility     string
	PushPermission *bool
}

type MirrorProviderReader interface {
	Repository(context.Context, config.Endpoint) (MirrorProviderRepository, ReadObservation, error)
}

type DefaultMirrorProviderReader struct {
	GitHub GitHubReader
}

func (r DefaultMirrorProviderReader) Repository(ctx context.Context, endpoint config.Endpoint) (MirrorProviderRepository, ReadObservation, error) {
	identity, err := mirrorEndpointIdentity(endpoint)
	if err != nil {
		return MirrorProviderRepository{}, ReadObservation{}, err
	}
	if identity.Provider != "github" {
		return MirrorProviderRepository{}, ReadObservation{
			Available: false,
			Evidence: Evidence{
				Source:    "provider.unsupported",
				Reference: identity.Provider + ":" + identity.Path,
				Detail:    "provider metadata adapter is not implemented for mirror posture v1",
			},
		}, nil
	}
	if r.GitHub == nil {
		return MirrorProviderRepository{}, ReadObservation{}, fmt.Errorf("GitHub mirror provider reader is required")
	}
	repository, observation, err := r.GitHub.Repository(ctx, identity.Path)
	if err != nil || !observation.Available {
		return MirrorProviderRepository{}, observation, err
	}
	visibility := "public"
	if repository.Private {
		visibility = "private"
	}
	if repository.Archived {
		visibility = "archived"
	}
	return MirrorProviderRepository{
		DefaultBranch:  repository.DefaultBranch,
		Visibility:     visibility,
		PushPermission: repository.PushPermission,
	}, observation, nil
}

func NewMirrorInventory() MirrorInventory {
	return MirrorInventory{
		Kind:     MirrorInventoryKind,
		Version:  MirrorInventoryVersion,
		Repos:    []MirrorRepositoryFacts{},
		Evidence: []Evidence{},
	}
}

func CollectMirrorPosture(ctx context.Context, spec config.Spec, local MirrorLocalObserver, providers MirrorProviderReader) (MirrorInventory, error) {
	if local == nil {
		return MirrorInventory{}, fmt.Errorf("mirror local observer is required")
	}
	if providers == nil {
		return MirrorInventory{}, fmt.Errorf("mirror provider reader is required")
	}

	inventory := NewMirrorInventory()
	for _, repo := range spec.Repos {
		facts, err := collectMirrorRepository(ctx, repo, local, providers)
		if err != nil {
			return MirrorInventory{}, fmt.Errorf("collect mirror posture for repo %q: %w", repo.ID, err)
		}
		inventory.Repos = append(inventory.Repos, facts)
	}
	return inventory, nil
}

func collectMirrorRepository(ctx context.Context, repo config.Repo, local MirrorLocalObserver, providers MirrorProviderReader) (MirrorRepositoryFacts, error) {
	canonicalIdentity, err := mirrorEndpointIdentity(repo.Canonical)
	if err != nil {
		return MirrorRepositoryFacts{}, fmt.Errorf("canonical identity: %w", err)
	}
	observation, err := local.Observe(repo)
	if err != nil {
		return MirrorRepositoryFacts{}, fmt.Errorf("observe default-branch reconciliation state: %w", err)
	}
	if len(observation.Status.Mirrors) != len(repo.Mirrors) || len(observation.MirrorBranches) != len(repo.Mirrors) {
		return MirrorRepositoryFacts{}, fmt.Errorf("mirror observation count does not match declared topology")
	}

	modeEvidence := Evidence{Source: "repora.config", Reference: repo.DurableID(), Detail: "declared repository mode"}
	facts := MirrorRepositoryFacts{
		ID:        repo.ID,
		UID:       repo.DurableID(),
		Mode:      Observed(repo.Mode, modeEvidence),
		Direction: Observed("canonical_to_mirror", Evidence{Source: "repora.mirror_semantics", Reference: repo.DurableID()}),
		Canonical: MirrorCanonicalFacts{Identity: canonicalIdentity},
		Mirrors:   []MirrorTargetFacts{},
		Evidence:  []Evidence{modeEvidence},
	}
	facts.Canonical.DefaultBranch = branchFact(observation.CanonicalBranch)
	facts.Canonical.Commit = commitFact(observation.Status.Canonical.Commit, Evidence{Source: "git.reconciliation", Reference: repo.DurableID() + ":canonical/HEAD"})

	canonicalProvider, canonicalProviderObs, err := providers.Repository(ctx, repo.Canonical)
	if err != nil {
		return MirrorRepositoryFacts{}, fmt.Errorf("read canonical provider metadata: %w", err)
	}
	facts.Canonical.Visibility = providerVisibilityFact(canonicalProvider, canonicalProviderObs)
	facts.Canonical.CurrentActorPushPermission = providerPushFact(canonicalProvider, canonicalProviderObs)
	if !observation.CanonicalBranch.Available && canonicalProviderObs.Available && canonicalProvider.DefaultBranch != "" {
		facts.Canonical.DefaultBranch = Observed(canonicalProvider.DefaultBranch, canonicalProviderObs.Evidence)
	}

	for i, endpoint := range repo.Mirrors {
		identity, err := mirrorEndpointIdentity(endpoint)
		if err != nil {
			return MirrorRepositoryFacts{}, fmt.Errorf("mirror[%d] identity: %w", i, err)
		}
		observed := observation.Status.Mirrors[i]
		branch := observation.MirrorBranches[i]
		providerRepo, providerObs, err := providers.Repository(ctx, endpoint)
		if err != nil {
			return MirrorRepositoryFacts{}, fmt.Errorf("read mirror[%d] provider metadata: %w", i, err)
		}
		branchValue := branchFact(branch)
		if !branch.Available && providerObs.Available && providerRepo.DefaultBranch != "" {
			branchValue = Observed(providerRepo.DefaultBranch, providerObs.Evidence)
		}
		mirror := MirrorTargetFacts{
			Identity:                   identity,
			CacheRemote:                Observed("mirror-"+strconv.Itoa(i), Evidence{Source: "repora.cache_remote", Reference: repo.DurableID()}),
			DefaultBranch:              branchValue,
			Commit:                     commitFact(observed.Commit, Evidence{Source: "git.reconciliation", Reference: observed.Target + ":HEAD"}),
			Divergence:                 divergenceFact(observed),
			Ahead:                      countFact(observed, true),
			Behind:                     countFact(observed, false),
			Visibility:                 providerVisibilityFact(providerRepo, providerObs),
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
	commit = strings.TrimSpace(commit)
	if commit == "" {
		return Unavailable[string](evidenceWithDetail(evidence, "default-branch commit unavailable"))
	}
	return Observed(commit, evidence)
}

func divergenceFact(observed status.MirrorResult) Fact[string] {
	evidence := Evidence{Source: "git.reconciliation", Reference: observed.Target + ":HEAD"}
	if observed.State == status.StateError {
		return Unavailable[string](evidenceWithDetail(evidence, "mirror reconciliation observation unavailable"))
	}
	return Observed(string(observed.State), evidence)
}

func countFact(observed status.MirrorResult, ahead bool) Fact[int] {
	evidence := Evidence{Source: "git.reconciliation", Reference: observed.Target + ":HEAD"}
	if observed.State == status.StateError {
		return Unavailable[int](evidenceWithDetail(evidence, "mirror reconciliation count unavailable"))
	}
	if ahead {
		return Observed(observed.Ahead, evidence)
	}
	return Observed(observed.Behind, evidence)
}

func branchDriftFact(canonical, mirror Fact[string]) Fact[bool] {
	if canonical.State != StateObserved || canonical.Value == nil || mirror.State != StateObserved || mirror.Value == nil {
		evidence := append(cloneEvidence(canonical.Evidence), mirror.Evidence...)
		if canonical.State == StateUnavailable || mirror.State == StateUnavailable {
			return Unavailable[bool](evidence...)
		}
		return Unknown[bool](evidence...)
	}
	return Observed(*canonical.Value != *mirror.Value, append(cloneEvidence(canonical.Evidence), mirror.Evidence...)...)
}

func providerVisibilityFact(repository MirrorProviderRepository, observation ReadObservation) Fact[string] {
	if !observation.Available {
		return Unavailable[string](observation.Evidence)
	}
	if strings.TrimSpace(repository.Visibility) == "" {
		return Unknown[string](observation.Evidence)
	}
	return Observed(strings.TrimSpace(repository.Visibility), observation.Evidence)
}

func providerPushFact(repository MirrorProviderRepository, observation ReadObservation) Fact[bool] {
	if !observation.Available {
		return Unavailable[bool](observation.Evidence)
	}
	if repository.PushPermission == nil {
		return Unknown[bool](observation.Evidence)
	}
	return Observed(*repository.PushPermission, observation.Evidence)
}

func mirrorEndpointIdentity(endpoint config.Endpoint) (MirrorEndpointIdentity, error) {
	provider := strings.TrimSpace(endpoint.Provider)
	pathValue := strings.Trim(strings.TrimSpace(endpoint.Path), "/")
	if pathValue == "" {
		legacyPath, err := mirrorLegacyPath(endpoint.URL)
		if err != nil {
			return MirrorEndpointIdentity{}, err
		}
		pathValue = legacyPath
	}
	if provider == "" || pathValue == "" {
		return MirrorEndpointIdentity{}, fmt.Errorf("provider and path are required")
	}
	return MirrorEndpointIdentity{Provider: provider, Path: pathValue}, nil
}

func mirrorLegacyPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	var pathValue string
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("parse legacy URL")
		}
		pathValue = parsed.Path
	} else if at := strings.LastIndex(raw, "@"); at >= 0 {
		if colon := strings.Index(raw[at+1:], ":"); colon >= 0 {
			pathValue = raw[at+1+colon+1:]
		}
	}
	pathValue = strings.TrimSuffix(strings.Trim(strings.TrimSpace(pathValue), "/"), ".git")
	parts := strings.Split(pathValue, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("legacy URL does not contain a safe repository path")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `\\:@?#`) {
			return "", fmt.Errorf("legacy URL contains an unsafe repository path")
		}
	}
	return pathValue, nil
}

func (i MirrorInventory) Validate() error {
	if i.Kind != MirrorInventoryKind || i.Version != MirrorInventoryVersion {
		return fmt.Errorf("unsupported mirror posture contract: kind=%q version=%d", i.Kind, i.Version)
	}
	if i.Repos == nil || i.Evidence == nil {
		return fmt.Errorf("mirror posture repos and evidence arrays are required")
	}
	for repoIndex, repo := range i.Repos {
		if strings.TrimSpace(repo.ID) == "" || strings.TrimSpace(repo.UID) == "" {
			return fmt.Errorf("repo[%d] id and uid are required", repoIndex)
		}
		if repo.Mirrors == nil || repo.Evidence == nil {
			return fmt.Errorf("repo[%d] mirror and evidence arrays are required", repoIndex)
		}
		if err := validateFact("mode", repo.Mode); err != nil {
			return err
		}
		if err := validateFact("direction", repo.Direction); err != nil {
			return err
		}
		if err := validateMirrorEndpointFacts("canonical", repo.Canonical.DefaultBranch, repo.Canonical.Commit, repo.Canonical.Visibility, repo.Canonical.CurrentActorPushPermission); err != nil {
			return err
		}
		for mirrorIndex, mirror := range repo.Mirrors {
			if strings.TrimSpace(mirror.Identity.Provider) == "" || strings.TrimSpace(mirror.Identity.Path) == "" {
				return fmt.Errorf("repo[%d] mirror[%d] identity is required", repoIndex, mirrorIndex)
			}
			for name, err := range map[string]error{
				"cache_remote":                  validateFact("cache_remote", mirror.CacheRemote),
				"default_branch":                validateFact("default_branch", mirror.DefaultBranch),
				"default_branch_drift":          validateFact("default_branch_drift", mirror.DefaultBranchDrift),
				"commit":                        validateFact("commit", mirror.Commit),
				"divergence":                    validateFact("divergence", mirror.Divergence),
				"ahead":                         validateFact("ahead", mirror.Ahead),
				"behind":                        validateFact("behind", mirror.Behind),
				"visibility":                    validateFact("visibility", mirror.Visibility),
				"current_actor_push_permission": validateFact("current_actor_push_permission", mirror.CurrentActorPushPermission),
				"tag_drift":                     validateFact("tag_drift", mirror.TagDrift),
				"release_drift":                 validateFact("release_drift", mirror.ReleaseDrift),
			} {
				if err != nil {
					return fmt.Errorf("repo[%d] mirror[%d] %s: %w", repoIndex, mirrorIndex, name, err)
				}
			}
		}
	}
	return nil
}

func validateMirrorEndpointFacts(prefix string, branch Fact[string], commit Fact[string], visibility Fact[string], push Fact[bool]) error {
	for name, err := range map[string]error{
		"default_branch":                validateFact("default_branch", branch),
		"commit":                        validateFact("commit", commit),
		"visibility":                    validateFact("visibility", visibility),
		"current_actor_push_permission": validateFact("current_actor_push_permission", push),
	} {
		if err != nil {
			return fmt.Errorf("%s %s: %w", prefix, name, err)
		}
	}
	return nil
}

func (i MirrorInventory) Marshal() ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode mirror posture inventory: %w", err)
	}
	return append(data, '\n'), nil
}
