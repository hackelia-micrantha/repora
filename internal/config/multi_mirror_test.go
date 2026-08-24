package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAllowsMultipleProviderPathMirrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`repos:
  - id: payments-api
    canonical:
      provider: gitlab
      path: org/payments-api
    mirrors:
      - provider: github
        path: org/payments-api
      - provider: bitbucket
        path: workspace/payments-api
`))

	spec, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got := len(spec.Repos[0].Mirrors); got != 2 {
		t.Fatalf("mirror count = %d, want 2", got)
	}
	if got := spec.Repos[0].Mirrors[1]; got.Provider != "bitbucket" || got.Path != "workspace/payments-api" {
		t.Fatalf("second mirror = %#v", got)
	}
}

func TestLoadRejectsDuplicateProviderPathMirrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`repos:
  - id: payments-api
    canonical:
      provider: gitlab
      path: org/payments-api
    mirrors:
      - provider: github
        path: org/payments-api
      - provider: github
        path: org/payments-api
`))

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "duplicate mirror target") {
		t.Fatalf("error = %v, want duplicate target rejection", err)
	}
}

func TestLoadRejectsLegacyURLInMultipleMirrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`repos:
  - id: payments-api
    canonical:
      provider: gitlab
      path: org/payments-api
    mirrors:
      - provider: github
        path: org/payments-api
      - provider: gitlab
        url: git@gitlab.com:backup/payments-api.git
`))

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "provider/path mirrors") {
		t.Fatalf("error = %v, want provider/path requirement", err)
	}
}
