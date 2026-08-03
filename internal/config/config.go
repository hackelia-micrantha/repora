package config

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"repoctl/internal/refpolicy"
)

type Spec struct {
	Repos []Repo `json:"repos" yaml:"repos"`
}

type Repo struct {
	ID        string           `json:"id" yaml:"id"`
	UID       string           `json:"uid" yaml:"uid"`
	Canonical Endpoint         `json:"canonical" yaml:"canonical"`
	Mirrors   []Endpoint       `json:"mirrors" yaml:"mirrors"`
	Mode      string           `json:"mode" yaml:"mode"`
	Policy    RepositoryPolicy `json:"policy,omitempty" yaml:"policy,omitempty"`
}

type RepositoryPolicy struct {
	Refs refpolicy.Policy `json:"refs,omitempty" yaml:"refs,omitempty"`
}

type Endpoint struct {
	Provider string `json:"provider" yaml:"provider"`
	Path     string `json:"path,omitempty" yaml:"path,omitempty"`
	URL      string `json:"url,omitempty" yaml:"url,omitempty"`
}

func (r Repo) DurableID() string {
	if r.UID != "" {
		return r.UID
	}
	return r.ID
}

func (r Repo) EffectiveRefPolicy() (refpolicy.Policy, error) {
	return r.Policy.Refs.Normalize()
}

func Load(path string) (Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Spec{}, fmt.Errorf("read config: %w", err)
	}

	spec, err := parse(data)
	if err != nil {
		return Spec{}, err
	}
	if err := validate(spec); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

func parse(data []byte) (Spec, error) {
	var spec Spec
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&spec); err != nil {
		return Spec{}, fmt.Errorf("parse config: %w", err)
	}
	return spec, nil
}

func validate(spec Spec) error {
	if len(spec.Repos) == 0 {
		return fmt.Errorf("SCHEMA-0001 requires at least one repo")
	}
	seenIDs := make(map[string]struct{}, len(spec.Repos))
	seenUIDs := make(map[string]struct{}, len(spec.Repos))
	for i, repo := range spec.Repos {
		repo.ID = strings.TrimSpace(repo.ID)
		repo.UID = strings.TrimSpace(repo.UID)
		if repo.ID == "" {
			return fmt.Errorf("repo id is required for repos[%d]", i)
		}
		if repo.UID == "" {
			repo.UID = repo.ID
		}
		if _, ok := seenIDs[repo.ID]; ok {
			return fmt.Errorf("duplicate repo id %q", repo.ID)
		}
		seenIDs[repo.ID] = struct{}{}
		if _, ok := seenUIDs[repo.UID]; ok {
			return fmt.Errorf("duplicate repo uid %q", repo.UID)
		}
		seenUIDs[repo.UID] = struct{}{}
		if err := validateEndpoint(repo.Canonical, "canonical", repo.ID); err != nil {
			return err
		}
		if repo.Canonical.Provider != "gitlab" {
			return fmt.Errorf("unsupported canonical provider %q for repo %q: only gitlab is supported", repo.Canonical.Provider, repo.ID)
		}
		if len(repo.Mirrors) != 1 {
			return fmt.Errorf("SCHEMA-0001 requires exactly one mirror for repo %q, got %d", repo.ID, len(repo.Mirrors))
		}
		if err := validateEndpoint(repo.Mirrors[0], "mirror", repo.ID); err != nil {
			return err
		}
		if !isSupportedMirrorProvider(repo.Mirrors[0].Provider) {
			return fmt.Errorf("unsupported mirror provider %q for repo %q: supported providers are github and gitlab", repo.Mirrors[0].Provider, repo.ID)
		}
		if repo.Mode == "" {
			repo.Mode = "mirror"
		}
		if repo.Mode != "mirror" {
			return fmt.Errorf("unsupported mode %q for repo %q: only mirror is supported", repo.Mode, repo.ID)
		}
		policy, err := repo.EffectiveRefPolicy()
		if err != nil {
			return fmt.Errorf("invalid ref policy for repo %q: %w", repo.ID, err)
		}
		repo.Policy.Refs = policy
		spec.Repos[i] = repo
	}
	return nil
}

func validateEndpoint(endpoint Endpoint, role, repoID string) error {
	if strings.TrimSpace(endpoint.Provider) == "" {
		return fmt.Errorf("%s provider is required for repo %q", role, repoID)
	}
	path := strings.TrimSpace(endpoint.Path)
	rawURL := strings.TrimSpace(endpoint.URL)
	pathSet := path != ""
	urlSet := rawURL != ""
	if pathSet == urlSet {
		return fmt.Errorf("%s must define exactly one of path or legacy url for repo %q", role, repoID)
	}
	if urlSet && containsURLCredentials(rawURL) {
		return fmt.Errorf("%s legacy url must not contain credentials for repo %q", role, repoID)
	}
	if pathSet {
		if strings.Contains(path, "://") || strings.Contains(path, "@") || strings.HasPrefix(path, "/") || strings.HasPrefix(path, ".") {
			return fmt.Errorf("%s path must be provider-relative for repo %q", role, repoID)
		}
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) < 2 {
			return fmt.Errorf("%s path must include an owner or namespace for repo %q", role, repoID)
		}
		for _, part := range parts {
			if part == "" || part == "." || part == ".." {
				return fmt.Errorf("%s path contains an invalid segment for repo %q", role, repoID)
			}
		}
	}
	return nil
}

func containsURLCredentials(raw string) bool {
	if strings.HasPrefix(raw, "git@") {
		return false
	}
	parsed, err := url.Parse(raw)
	return err == nil && parsed.User != nil
}

func isSupportedMirrorProvider(provider string) bool {
	return provider == "github" || provider == "gitlab"
}
