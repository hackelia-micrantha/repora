package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Spec struct {
	Repos []Repo `json:"repos" yaml:"repos"`
}

type Repo struct {
	ID        string     `json:"id" yaml:"id"`
	Canonical Endpoint   `json:"canonical" yaml:"canonical"`
	Mirrors   []Endpoint `json:"mirrors" yaml:"mirrors"`
	Mode      string     `json:"mode" yaml:"mode"`
}

type Endpoint struct {
	Provider string `json:"provider" yaml:"provider"`
	URL      string `json:"url" yaml:"url"`
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
	for i, repo := range spec.Repos {
		if repo.ID == "" {
			return fmt.Errorf("repo id is required for repos[%d]", i)
		}
		if _, ok := seenIDs[repo.ID]; ok {
			return fmt.Errorf("duplicate repo id %q", repo.ID)
		}
		seenIDs[repo.ID] = struct{}{}
		if repo.Canonical.Provider == "" || repo.Canonical.URL == "" {
			return fmt.Errorf("canonical provider and url are required for repo %q", repo.ID)
		}
		if repo.Canonical.Provider != "gitlab" {
			return fmt.Errorf("unsupported canonical provider %q for repo %q: only gitlab is supported", repo.Canonical.Provider, repo.ID)
		}
		if len(repo.Mirrors) != 1 {
			return fmt.Errorf("SCHEMA-0001 requires exactly one mirror for repo %q, got %d", repo.ID, len(repo.Mirrors))
		}
		if repo.Mirrors[0].Provider == "" || repo.Mirrors[0].URL == "" {
			return fmt.Errorf("mirror provider and url are required for repo %q", repo.ID)
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
		spec.Repos[i] = repo
	}
	return nil
}

func isSupportedMirrorProvider(provider string) bool {
	return provider == "github" || provider == "gitlab"
}
