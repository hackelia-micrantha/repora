package config

import (
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
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return Spec{}, fmt.Errorf("parse config: %w", err)
	}
	return spec, nil
}

func validate(spec Spec) error {
	if len(spec.Repos) != 1 {
		return fmt.Errorf("SCHEMA-0001 requires exactly one repo, got %d", len(spec.Repos))
	}
	repo := spec.Repos[0]
	if repo.ID == "" {
		return fmt.Errorf("repo id is required")
	}
	if repo.Canonical.Provider == "" || repo.Canonical.URL == "" {
		return fmt.Errorf("canonical provider and url are required")
	}
	if len(repo.Mirrors) != 1 {
		return fmt.Errorf("SCHEMA-0001 requires exactly one mirror, got %d", len(repo.Mirrors))
	}
	if repo.Mirrors[0].Provider == "" || repo.Mirrors[0].URL == "" {
		return fmt.Errorf("mirror provider and url are required")
	}
	if repo.Mode == "" {
		repo.Mode = "mirror"
	}
	if repo.Mode != "mirror" {
		return fmt.Errorf("unsupported mode %q: only mirror is supported", repo.Mode)
	}
	spec.Repos[0] = repo
	return nil
}
