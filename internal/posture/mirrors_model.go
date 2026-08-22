package posture

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"repoctl/internal/config"
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
	DefaultBranchDrift         Fact[bool]              `json:"default_branch_drift"`
	Commit                     Fact[string]           `json:"commit"`
	Divergence                 Fact[string]           `json:"divergence"`
	Ahead                      Fact[int]              `json:"ahead"`
	Behind                     Fact[int]              `json:"behind"`
	Visibility                 Fact[string]           `json:"visibility"`
	CurrentActorPushPermission Fact[bool]             `json:"current_actor_push_permission"`
	TagDrift                   Fact[bool]              `json:"tag_drift"`
	ReleaseDrift               Fact[bool]              `json:"release_drift"`
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

func NewMirrorInventory() MirrorInventory {
	return MirrorInventory{Kind: MirrorInventoryKind, Version: MirrorInventoryVersion, Repos: []MirrorRepositoryFacts{}, Evidence: []Evidence{}}
}

func mirrorEndpointIdentity(endpoint config.Endpoint) (MirrorEndpointIdentity, error) {
	provider := strings.TrimSpace(endpoint.Provider)
	pathValue := strings.Trim(strings.TrimSpace(endpoint.Path), "/")
	if pathValue == "" {
		legacy, err := mirrorLegacyPath(endpoint.URL)
		if err != nil {
			return MirrorEndpointIdentity{}, err
		}
		pathValue = legacy
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
	for ri, repo := range i.Repos {
		if strings.TrimSpace(repo.ID) == "" || strings.TrimSpace(repo.UID) == "" {
			return fmt.Errorf("repo[%d] id and uid are required", ri)
		}
		if repo.Mirrors == nil || repo.Evidence == nil {
			return fmt.Errorf("repo[%d] mirror and evidence arrays are required", ri)
		}
		if strings.TrimSpace(repo.Canonical.Identity.Provider) == "" || strings.TrimSpace(repo.Canonical.Identity.Path) == "" {
			return fmt.Errorf("repo[%d] canonical identity is required", ri)
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
		for mi, mirror := range repo.Mirrors {
			if strings.TrimSpace(mirror.Identity.Provider) == "" || strings.TrimSpace(mirror.Identity.Path) == "" {
				return fmt.Errorf("repo[%d] mirror[%d] identity is required", ri, mi)
			}
			checks := []error{
				validateFact("cache_remote", mirror.CacheRemote), validateFact("default_branch", mirror.DefaultBranch), validateFact("default_branch_drift", mirror.DefaultBranchDrift),
				validateFact("commit", mirror.Commit), validateFact("divergence", mirror.Divergence), validateFact("ahead", mirror.Ahead), validateFact("behind", mirror.Behind),
				validateFact("visibility", mirror.Visibility), validateFact("current_actor_push_permission", mirror.CurrentActorPushPermission), validateFact("tag_drift", mirror.TagDrift), validateFact("release_drift", mirror.ReleaseDrift),
			}
			for _, err := range checks {
				if err != nil {
					return fmt.Errorf("repo[%d] mirror[%d]: %w", ri, mi, err)
				}
			}
		}
	}
	return nil
}

func validateMirrorEndpointFacts(prefix string, branch, commit, visibility Fact[string], push Fact[bool]) error {
	for _, err := range []error{validateFact("default_branch", branch), validateFact("commit", commit), validateFact("visibility", visibility), validateFact("current_actor_push_permission", push)} {
		if err != nil {
			return fmt.Errorf("%s: %w", prefix, err)
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
