package posture

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"repoctl/internal/config"
	"repoctl/internal/status"
)

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
	observation, err := local.Observe(repo)
	if err != nil {
		return MirrorRepositoryFacts{}, fmt.Errorf("observe reconciliation state: %w", err)
	}
	if len(observation.Status.Mirrors) != len(repo.Mirrors) || len(observation.MirrorBranches) != len(repo.Mirrors) {
		return MirrorRepositoryFacts{}, fmt.Errorf("mirror observation count does not match declared topology")
	}
	configEvidence := Evidence{Source: "repora.config", Reference: repo.DurableID()}
	facts := MirrorRepositoryFacts{
		ID:        repo.ID,
		UID:       repo.DurableID(),
		Mode:      Observed(repo.Mode, configEvidence),
		Direction: Observed("canonical_to_mirror", Evidence{Source: "repora.mirror_semantics", Reference: repo.DurableID()}),
		Canonical: MirrorCanonicalFacts{Identity: canonicalIdentity},
		Mirrors:   []MirrorTargetFacts{},
		Evidence:  []Evidence{configEvidence},
	}
	facts.Canonical.DefaultBranch = branchFact(observation.CanonicalBranch)
	facts.Canonical.Commit = commitFact(
		observation.Status.Canonical.Commit,
		Evidence{Source: "git.reconciliation", Reference: repo.DurableID() + ":canonical/HEAD"},
	)
	canonicalProvider, canonicalProviderObservation, err := providers.Repository(ctx, repo.Canonical)
	if err != nil {
		return MirrorRepositoryFacts{}, fmt.Errorf("read canonical provider metadata: %w", err)
	}
	facts.Canonical.Visibility = providerVisibilityFact(canonicalProvider, canonicalProviderObservation)
	facts.Canonical.CurrentActorPushPermission = providerPushFact(canonicalProvider, canonicalProviderObservation)
	if facts.Canonical.DefaultBranch.State != StateObserved && canonicalProviderObservation.Available && canonicalProvider.DefaultBranch != "" {
		facts.Canonical.DefaultBranch = Observed(canonicalProvider.DefaultBranch, canonicalProviderObservation.Evidence)
	}

	for index, endpoint := range repo.Mirrors {
		mirror, err := collectMirrorTarget(ctx, repo, index, endpoint, facts.Canonical.DefaultBranch, observation, providers)
		if err != nil {
			return MirrorRepositoryFacts{}, err
		}
		facts.Mirrors = append(facts.Mirrors, mirror)
	}
	return facts, nil
}

func collectMirrorTarget(
	ctx context.Context,
	repo config.Repo,
	index int,
	endpoint config.Endpoint,
	canonicalBranch Fact[string],
	observation MirrorLocalObservation,
	providers MirrorProviderReader,
) (MirrorTargetFacts, error) {
	identity, err := mirrorEndpointIdentity(endpoint)
	if err != nil {
		return MirrorTargetFacts{}, fmt.Errorf("mirror[%d] identity: %w", index, err)
	}
	statusResult := observation.Status.Mirrors[index]
	providerRepository, providerObservation, err := providers.Repository(ctx, endpoint)
	if err != nil {
		return MirrorTargetFacts{}, fmt.Errorf("read mirror[%d] provider metadata: %w", index, err)
	}
	defaultBranch := branchFact(observation.MirrorBranches[index])
	if defaultBranch.State != StateObserved && providerObservation.Available && providerRepository.DefaultBranch != "" {
		defaultBranch = Observed(providerRepository.DefaultBranch, providerObservation.Evidence)
	}
	mirror := MirrorTargetFacts{
		Identity:      identity,
		CacheRemote:   Observed("mirror-"+strconv.Itoa(index), Evidence{Source: "repora.cache_remote", Reference: repo.DurableID()}),
		DefaultBranch: defaultBranch,
		Commit: commitFact(
			statusResult.Commit,
			Evidence{Source: "git.reconciliation", Reference: identity.Provider + ":" + identity.Path + ":HEAD"},
		),
		Divergence:                 divergenceFact(statusResult),
		Ahead:                      countFact(statusResult, true),
		Behind:                     countFact(statusResult, false),
		Visibility:                 providerVisibilityFact(providerRepository, providerObservation),
		CurrentActorPushPermission: providerPushFact(providerRepository, providerObservation),
		TagDrift: Unknown[bool](Evidence{
			Source:    "repora.ref_scope",
			Reference: identity.Provider + ":" + identity.Path,
			Detail:    "tag drift is outside mirror posture v1 default-branch scope",
		}),
		ReleaseDrift: Unknown[bool](Evidence{
			Source:    "repora.ref_scope",
			Reference: identity.Provider + ":" + identity.Path,
			Detail:    "release drift requires provider release adapters not implemented in v1",
		}),
	}
	mirror.DefaultBranchDrift = branchDriftFact(canonicalBranch, mirror.DefaultBranch)
	return mirror, nil
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
	if ahead {
		return Observed(result.Ahead, evidence)
	}
	return Observed(result.Behind, evidence)
}

func branchDriftFact(canonical, mirror Fact[string]) Fact[bool] {
	evidence := append(cloneEvidence(canonical.Evidence), mirror.Evidence...)
	if canonical.State != StateObserved || canonical.Value == nil || mirror.State != StateObserved || mirror.Value == nil {
		if canonical.State == StateUnavailable || mirror.State == StateUnavailable {
			return Unavailable[bool](evidence...)
		}
		return Unknown[bool](evidence...)
	}
	return Observed(*canonical.Value != *mirror.Value, evidence...)
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
