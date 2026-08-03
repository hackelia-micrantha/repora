package config

import (
	"path/filepath"
	"strings"
	"testing"

	"repoctl/internal/refpolicy"
)

func TestLoadDefaultsClosedRefPolicy(t *testing.T) {
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
	got, err := spec.Repos[0].EffectiveRefPolicy()
	if err != nil {
		t.Fatalf("EffectiveRefPolicy returned error: %v", err)
	}
	if got != refpolicy.Default() {
		t.Fatalf("policy = %#v, want %#v", got, refpolicy.Default())
	}
}

func TestLoadAcceptsExplicitClosedRefPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`repos:
  - id: payments-api
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/payments-api.git
    mirrors:
      - provider: github
        url: git@github.com:org/payments-api.git
    policy:
      refs:
        version: 1
        scope: default-branch-only
        destructive: require-force
`))

	if _, err := Load(path); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
}

func TestLoadRejectsUnsupportedRefPolicy(t *testing.T) {
	tests := []struct {
		name  string
		field string
		value string
		want  string
	}{
		{name: "version", field: "version", value: "2", want: "version"},
		{name: "scope", field: "scope", value: "all-branches", want: "scope"},
		{name: "destructive", field: "destructive", value: "allow", want: "destructive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "repora.yaml")
			policy := "        version: 1\n        scope: default-branch-only\n        destructive: require-force\n"
			policy = strings.Replace(policy, "        "+tt.field+": "+map[string]string{"version": "1", "scope": "default-branch-only", "destructive": "require-force"}[tt.field], "        "+tt.field+": "+tt.value, 1)
			writeFile(t, path, []byte(`repos:
  - id: payments-api
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/payments-api.git
    mirrors:
      - provider: github
        url: git@github.com:org/payments-api.git
    policy:
      refs:
`+policy))
			_, err := Load(path)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}
