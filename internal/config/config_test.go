package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAcceptsProviderPathSchemaV1(t *testing.T) {
	spec, err := Load(filepath.Join("..", "..", "testdata", "repora.yaml"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if spec.Version != 1 {
		t.Fatalf("version = %d, want 1", spec.Version)
	}
	if spec.DefaultTransport != "https" {
		t.Fatalf("default transport = %q, want https", spec.DefaultTransport)
	}
	if len(spec.Repos) != 4 {
		t.Fatalf("repo count = %d, want 4", len(spec.Repos))
	}

	repo := spec.Repos[0]
	if repo.ID != "anthesis" {
		t.Fatalf("repo ID = %q, want anthesis", repo.ID)
	}
	if repo.UID != "repo.micrantha.anthesis" {
		t.Fatalf("repo UID = %q", repo.UID)
	}
	if repo.Canonical.Provider != "gitlab" {
		t.Fatalf("canonical provider = %q", repo.Canonical.Provider)
	}
	if repo.Canonical.Path != "micrantha/anthesis" {
		t.Fatalf("canonical path = %q", repo.Canonical.Path)
	}
	if repo.Canonical.URL != "https://gitlab.com/micrantha/anthesis.git" {
		t.Fatalf("canonical URL = %q", repo.Canonical.URL)
	}
	if len(repo.Mirrors) != 2 {
		t.Fatalf("mirror count = %d, want 2", len(repo.Mirrors))
	}
	if repo.Mirrors[0].URL != "https://github.com/hackelia-micrantha/anthesis.git" {
		t.Fatalf("mirror URL = %q", repo.Mirrors[0].URL)
	}
}

func TestLoadDefaultsTransportToHTTPS(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`
version: 1
providers:
  gitlab:
    base_urls:
      https: https://gitlab.com
      ssh: git@gitlab.com:
  github:
    base_urls:
      https: https://github.com
      ssh: git@github.com:
repos:
  - id: blog
    uid: repo.ryjen.blog
    canonical:
      provider: gitlab
      path: ryjen/blog
    mirrors:
      - provider: github
        path: ryjen/blog
    mode: mirror
`))

	spec, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got := spec.DefaultTransport; got != "https" {
		t.Fatalf("default transport = %q, want https", got)
	}
}

func TestLoadRejectsDuplicateRepoUID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`
version: 1
providers:
  gitlab:
    base_urls:
      https: https://gitlab.com
      ssh: git@gitlab.com:
  github:
    base_urls:
      https: https://github.com
      ssh: git@github.com:
repos:
  - id: anthesis
    uid: repo.shared
    canonical:
      provider: gitlab
      path: micrantha/anthesis
    mirrors:
      - provider: github
        path: hackelia-micrantha/anthesis
    mode: mirror
  - id: hyperion
    uid: repo.shared
    canonical:
      provider: gitlab
      path: micrantha/laboratory/hyperion
    mirrors:
      - provider: github
        path: hackelia-micrantha/hyperion
    mode: mirror
`))

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load returned nil error, want duplicate uid rejection")
	}
}

func TestLoadRejectsUnknownProvider(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`
version: 1
providers:
  github:
    base_urls:
      https: https://github.com
      ssh: git@github.com:
repos:
  - id: anthesis
    uid: repo.micrantha.anthesis
    canonical:
      provider: gitlab
      path: micrantha/anthesis
    mirrors:
      - provider: github
        path: hackelia-micrantha/anthesis
    mode: mirror
`))

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load returned nil error, want unknown provider rejection")
	}
}

func TestLoadRejectsPathWithDotGitSuffix(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`
version: 1
providers:
  gitlab:
    base_urls:
      https: https://gitlab.com
      ssh: git@gitlab.com:
  github:
    base_urls:
      https: https://github.com
      ssh: git@github.com:
repos:
  - id: anthesis
    uid: repo.micrantha.anthesis
    canonical:
      provider: gitlab
      path: micrantha/anthesis.git
    mirrors:
      - provider: github
        path: hackelia-micrantha/anthesis
    mode: mirror
`))

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load returned nil error, want .git suffix rejection")
	}
}

func TestDeriveURLHTTPS(t *testing.T) {
	provider := Provider{
		BaseURLs: BaseURLs{
			HTTPS: "https://github.com",
			SSH:   "git@github.com:",
		},
	}

	got, err := DeriveURL(provider, "ryjen/dubnium", "https")
	if err != nil {
		t.Fatalf("DeriveURL returned error: %v", err)
	}

	want := "https://github.com/ryjen/dubnium.git"
	if got != want {
		t.Fatalf("url = %q, want %q", got, want)
	}
}

func TestDeriveURLSSH(t *testing.T) {
	provider := Provider{
		BaseURLs: BaseURLs{
			HTTPS: "https://gitlab.com",
			SSH:   "git@gitlab.com:",
		},
	}

	got, err := DeriveURL(provider, "micrantha/laboratory/dubnium", "ssh")
	if err != nil {
		t.Fatalf("DeriveURL returned error: %v", err)
	}

	want := "git@gitlab.com:micrantha/laboratory/dubnium.git"
	if got != want {
		t.Fatalf("url = %q, want %q", got, want)
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
