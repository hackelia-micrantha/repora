package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAcceptsProviderPaths(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`repos:
  - id: payments-api
    canonical:
      provider: gitlab
      path: org/platform/payments-api
    mirrors:
      - provider: github
        path: org/payments-api
`))

	spec, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Repos[0].Canonical.Path != "org/platform/payments-api" {
		t.Fatalf("canonical = %#v", spec.Repos[0].Canonical)
	}
	if spec.Repos[0].Canonical.URL != "" {
		t.Fatalf("config validation derived url %q", spec.Repos[0].Canonical.URL)
	}
}

func TestLoadRejectsAmbiguousEndpoint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`repos:
  - id: payments-api
    canonical:
      provider: gitlab
      path: org/payments-api
      url: https://gitlab.com/org/payments-api.git
    mirrors:
      - provider: github
        path: org/payments-api
`))

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "exactly one of path or legacy url") {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadRejectsCredentialBearingLegacyURLs(t *testing.T) {
	tests := []string{
		"https://token@github.com/org/payments-api.git",
		"https://user:password@github.com/org/payments-api.git",
	}

	for _, legacyURL := range tests {
		t.Run(legacyURL, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "repora.yaml")
			writeFile(t, path, []byte(`repos:
  - id: payments-api
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/payments-api.git
    mirrors:
      - provider: github
        url: `+legacyURL+`
`))

			_, err := Load(path)
			if err == nil || !strings.Contains(err.Error(), "must not contain credentials") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestLoadAcceptsSCPLegacyURL(t *testing.T) {
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

	if _, err := Load(path); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
}
