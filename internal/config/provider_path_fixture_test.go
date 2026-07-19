package config

import (
	"path/filepath"
	"testing"
)

func TestLoadPreferredProviderPathFixture(t *testing.T) {
	spec, err := Load(filepath.Join("..", "..", "testdata", "repora-provider-path.yaml"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(spec.Repos) != 1 {
		t.Fatalf("repo count = %d, want 1", len(spec.Repos))
	}

	repo := spec.Repos[0]
	if repo.ID != "payments-api" || repo.UID != "repo.org.payments-api" {
		t.Fatalf("identity = %q/%q", repo.ID, repo.UID)
	}
	if repo.Canonical.Provider != "gitlab" || repo.Canonical.Path != "org/payments-api" || repo.Canonical.URL != "" {
		t.Fatalf("canonical = %#v", repo.Canonical)
	}
	if len(repo.Mirrors) != 1 {
		t.Fatalf("mirror count = %d, want 1", len(repo.Mirrors))
	}
	mirror := repo.Mirrors[0]
	if mirror.Provider != "github" || mirror.Path != "org/payments-api" || mirror.URL != "" {
		t.Fatalf("mirror = %#v", mirror)
	}
	if repo.Mode != "mirror" {
		t.Fatalf("mode = %q, want mirror", repo.Mode)
	}
}
