package config

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"repoctl/internal/refpolicy"
)

var artifactValueKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9._-]*$`)

type Spec struct {
	Repos []Repo `json:"repos" yaml:"repos"`
}

type Repo struct {
	ID        string              `json:"id" yaml:"id"`
	UID       string              `json:"uid" yaml:"uid"`
	Canonical Endpoint            `json:"canonical" yaml:"canonical"`
	Mirrors   []Endpoint          `json:"mirrors" yaml:"mirrors"`
	Mode      string              `json:"mode" yaml:"mode"`
	Policy    RepositoryPolicy    `json:"policy,omitempty" yaml:"policy,omitempty"`
	Artifacts RepositoryArtifacts `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`
}

type RepositoryPolicy struct {
	Refs refpolicy.Policy `json:"refs,omitempty" yaml:"refs,omitempty"`
}

type RepositoryArtifacts struct {
	Readme *ReadmeArtifact `json:"readme,omitempty" yaml:"readme,omitempty"`
}

type ReadmeArtifact struct {
	Template string            `json:"template" yaml:"template"`
	Values   map[string]string `json:"values,omitempty" yaml:"values,omitempty"`
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
		if len(repo.Mirrors) == 0 {
			return fmt.Errorf("SCHEMA-0001 requires at least one mirror for repo %q", repo.ID)
		}
		seenMirrors := make(map[string]struct{}, len(repo.Mirrors))
		for j, mirror := range repo.Mirrors {
			if err := validateEndpoint(mirror, fmt.Sprintf("mirrors[%d]", j), repo.ID); err != nil {
				return err
			}
			if !isSupportedMirrorProvider(mirror.Provider) {
				return fmt.Errorf("unsupported mirror provider %q for repo %q: supported providers are github and gitlab", mirror.Provider, repo.ID)
			}
			if len(repo.Mirrors) > 1 && strings.TrimSpace(mirror.Path) == "" {
				return fmt.Errorf("repo %q requires provider/path mirrors when more than one mirror is configured", repo.ID)
			}
			if path := strings.Trim(strings.TrimSpace(mirror.Path), "/"); path != "" {
				key := strings.TrimSpace(mirror.Provider) + ":" + path
				if _, exists := seenMirrors[key]; exists {
					return fmt.Errorf("duplicate mirror target %q for repo %q", key, repo.ID)
				}
				seenMirrors[key] = struct{}{}
			}
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
		if err := validateArtifacts(&repo, i); err != nil {
			return err
		}
		spec.Repos[i] = repo
	}
	return nil
}

func validateArtifacts(repo *Repo, index int) error {
	readme := repo.Artifacts.Readme
	if readme == nil {
		return nil
	}
	if strings.TrimSpace(repo.Canonical.Path) == "" {
		return fmt.Errorf("README artifact for repo %q requires provider/path canonical identity", repo.ID)
	}

	readme.Template = strings.TrimSpace(readme.Template)
	if err := validateArtifactTemplatePath(readme.Template); err != nil {
		return fmt.Errorf("invalid README artifact template for repo %q at repos[%d]: %w", repo.ID, index, err)
	}
	keys := make([]string, 0, len(readme.Values))
	for key := range readme.Values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := readme.Values[key]
		if !artifactValueKeyPattern.MatchString(key) {
			return fmt.Errorf("invalid README artifact value key %q for repo %q", key, repo.ID)
		}
		if strings.ContainsRune(value, '\x00') {
			return fmt.Errorf("README artifact value %q for repo %q contains NUL", key, repo.ID)
		}
	}
	return nil
}

func validateArtifactTemplatePath(value string) error {
	if value == "" {
		return fmt.Errorf("template path is required")
	}
	if strings.ContainsAny(value, "\\:\x00\r\n\t") || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "~") {
		return fmt.Errorf("template path must be a portable configuration-root-relative path")
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != value {
		return fmt.Errorf("template path must not contain traversal or redundant segments")
	}
	for _, segment := range strings.Split(clean, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("template path contains an invalid segment")
		}
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
