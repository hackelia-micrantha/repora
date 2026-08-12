package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadReadsREADMEArtifactConfig(t *testing.T) {
	path := writeArtifactConfig(t, `    artifacts:
      readme:
        template: .repora/templates/README.md.tmpl
        values:
          title: Repora
          summary: Deterministic repository mirror management
`)

	spec, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	readme := spec.Repos[0].Artifacts.Readme
	if readme == nil {
		t.Fatal("README artifact config is nil")
	}
	if readme.Template != ".repora/templates/README.md.tmpl" {
		t.Fatalf("template = %q", readme.Template)
	}
	if readme.Values["title"] != "Repora" || readme.Values["summary"] == "" {
		t.Fatalf("values = %#v", readme.Values)
	}
}

func TestLoadLeavesUnconfiguredRepositoryArtifactFree(t *testing.T) {
	path := writeArtifactConfig(t, "")
	spec, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if spec.Repos[0].Artifacts.Readme != nil {
		t.Fatalf("unconfigured README artifact = %#v, want nil", spec.Repos[0].Artifacts.Readme)
	}
}

func TestLoadRejectsUnknownArtifactType(t *testing.T) {
	path := writeArtifactConfig(t, `    artifacts:
      workflow:
        template: workflow.tmpl
`)
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "field workflow not found") {
		t.Fatalf("Load() error = %v, want strict unknown-artifact rejection", err)
	}
}

func TestLoadRejectsREADMEArtifactWithLegacyCanonicalURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	content := `repos:
  - id: repora
    uid: repo.repora
    canonical:
      provider: gitlab
      url: git@gitlab.com:micrantha/repora.git
    mirrors:
      - provider: github
        url: git@github.com:hackelia-micrantha/repora.git
    mode: mirror
    artifacts:
      readme:
        template: templates/README.md.tmpl
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "requires provider/path canonical identity") {
		t.Fatalf("Load() error = %v, want provider/path identity rejection", err)
	}
}

func TestLoadRejectsUnsafeREADMEArtifactTemplatePaths(t *testing.T) {
	for _, template := range []string{
		"",
		"/tmp/README.md.tmpl",
		"../README.md.tmpl",
		"templates/../README.md.tmpl",
		"./README.md.tmpl",
		"C:\\templates\\README.md.tmpl",
		"https://example.test/README.md.tmpl",
		"~/README.md.tmpl",
	} {
		t.Run(strings.ReplaceAll(template, "/", "_"), func(t *testing.T) {
			quoted := template
			if quoted == "" {
				quoted = `""`
			}
			path := writeArtifactConfig(t, "    artifacts:\n      readme:\n        template: "+quoted+"\n")
			_, err := Load(path)
			if err == nil || !strings.Contains(err.Error(), "invalid README artifact template") {
				t.Fatalf("Load(template=%q) error = %v, want template rejection", template, err)
			}
		})
	}
}

func TestLoadRejectsInvalidREADMEArtifactValueKey(t *testing.T) {
	path := writeArtifactConfig(t, `    artifacts:
      readme:
        template: templates/README.md.tmpl
        values:
          Bad-Key: value
`)
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "invalid README artifact value key") {
		t.Fatalf("Load() error = %v, want value-key rejection", err)
	}
}

func writeArtifactConfig(t *testing.T, artifactYAML string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "repora.yaml")
	content := `repos:
  - id: repora
    uid: repo.repora
    canonical:
      provider: gitlab
      path: micrantha/repora
    mirrors:
      - provider: github
        path: hackelia-micrantha/repora
    mode: mirror
` + artifactYAML
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
