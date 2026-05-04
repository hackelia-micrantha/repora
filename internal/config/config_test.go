package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsSingleMirrorRepo(t *testing.T) {
	spec, err := Load(filepath.Join("..", "..", "testdata", "repora.yaml"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(spec.Repos) != 1 {
		t.Fatalf("repo count = %d, want 1", len(spec.Repos))
	}

	repo := spec.Repos[0]
	if repo.ID != "payments-api" {
		t.Fatalf("repo ID = %q, want payments-api", repo.ID)
	}
	if repo.Canonical.Provider != "gitlab" || repo.Canonical.URL != "git@gitlab.com:org/payments-api.git" {
		t.Fatalf("canonical = %#v", repo.Canonical)
	}
	if len(repo.Mirrors) != 1 {
		t.Fatalf("mirror count = %d, want 1", len(repo.Mirrors))
	}
	if repo.Mirrors[0].Provider != "github" || repo.Mirrors[0].URL != "git@github.com:org/payments-api.git" {
		t.Fatalf("mirror = %#v", repo.Mirrors[0])
	}
	if repo.Mode != "mirror" {
		t.Fatalf("mode = %q, want mirror", repo.Mode)
	}
}

func TestLoadRejectsMoreThanOneMirror(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`repos:
  - id: payments-api
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/payments-api.git
    mirrors:
      - provider: github
        url: git@github.com:org/payments-api.git
      - provider: gitlab
        url: git@gitlab.com:backup/payments-api.git
    mode: mirror
`))

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load returned nil error, want rejection")
	}
}

func TestLoadAllowsMultipleRepos(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`repos:
  - id: payments-api
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/payments-api.git
    mirrors:
      - provider: github
        url: git@github.com:org/payments-api.git
    mode: mirror
  - id: auth-service
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/auth-service.git
    mirrors:
      - provider: github
        url: git@github.com:org/auth-service.git
    mode: mirror
`))

	spec, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(spec.Repos) != 2 {
		t.Fatalf("repo count = %d, want 2", len(spec.Repos))
	}
}

func TestLoadDefaultsModeToMirror(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`repos:
  - id: payments-api
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/payments-api.git
    mirrors:
      - provider: github
        url: git@github.com:org/payments-api.git
`))

	spec, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got := spec.Repos[0].Mode; got != "mirror" {
		t.Fatalf("mode = %q, want mirror", got)
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
