package config

import (
	"path/filepath"
	"testing"
)

func TestPackagedExampleConfigLoads(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "repora.yaml")
	spec, err := Load(path)
	if err != nil {
		t.Fatalf("load packaged example config: %v", err)
	}
	if len(spec.Repos) != 1 {
		t.Fatalf("example repo count = %d, want 1", len(spec.Repos))
	}

	repo := spec.Repos[0]
	if repo.ID != "example" || repo.UID != "example.example" {
		t.Fatalf("example identity = %q/%q", repo.ID, repo.UID)
	}
	if repo.Canonical.Provider != "gitlab" || repo.Canonical.Path != "example-group/example" || repo.Canonical.URL != "" {
		t.Fatalf("example canonical = %#v", repo.Canonical)
	}
	if len(repo.Mirrors) != 1 {
		t.Fatalf("example mirror count = %d, want 1", len(repo.Mirrors))
	}
	mirror := repo.Mirrors[0]
	if mirror.Provider != "github" || mirror.Path != "example-org/example" || mirror.URL != "" {
		t.Fatalf("example mirror = %#v", mirror)
	}
	if repo.Mode != "mirror" {
		t.Fatalf("example mode = %q, want mirror", repo.Mode)
	}
}
