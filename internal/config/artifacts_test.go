package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAcceptsREADMEArtifactConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`repos:
  - id: repora
    uid: repo.repora
    canonical:
      provider: gitlab
      path: micrantha/repora
    mirrors:
      - provider: github
        path: hackelia-micrantha/repora
    artifacts:
      readme:
        template: templates/README.md.tmpl
        values:
          title: Repora
          short_summary: Deterministic repository management
`))

	spec, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	readme := spec.Repos[0].Artifacts.README
	if readme == nil {
		t.Fatal("README artifact config is nil")
	}
	if readme.Template != "templates/README.md.tmpl" {
		t.Fatalf("template = %q", readme.Template)
	}
	if got := readme.Values["title"]; got != "Repora" {
		t.Fatalf("title = %q", got)
	}
}

func TestLoadRejectsREADMEWithLegacyCanonicalURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`repos:
  - id: repora
    canonical:
      provider: gitlab
      url: git@gitlab.com:micrantha/repora.git
    mirrors:
      - provider: github
        url: git@github.com:hackelia-micrantha/repora.git
    artifacts:
      readme:
        template: templates/README.md.tmpl
`))

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "requires provider/path canonical identity") {
		t.Fatalf("Load error = %v, want provider/path canonical requirement", err)
	}
}

func TestLoadRejectsUnknownArtifactType(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`repos:
  - id: repora
    canonical:
      provider: gitlab
      path: micrantha/repora
    mirrors:
      - provider: github
        path: hackelia-micrantha/repora
    artifacts:
      license:
        template: templates/LICENSE.tmpl
`))

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "field license not found") {
		t.Fatalf("Load error = %v, want unknown artifact rejection", err)
	}
}

func TestLoadRejectsREADMEWithoutTemplate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`repos:
  - id: repora
    canonical:
      provider: gitlab
      path: micrantha/repora
    mirrors:
      - provider: github
        path: hackelia-micrantha/repora
    artifacts:
      readme:
        values:
          title: Repora
`))

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "artifacts.readme.template is required") {
		t.Fatalf("Load error = %v, want missing template rejection", err)
	}
}

func TestLoadRejectsUnsafeREADMETemplatePaths(t *testing.T) {
	for _, template := range []string{
		"../README.md.tmpl",
		"templates/../README.md.tmpl",
		"/tmp/README.md.tmpl",
		"C:\\README.md.tmpl",
		"https://example.com/README.md.tmpl",
		"templates\\README.md.tmpl",
	} {
		t.Run(template, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "repora.yaml")
			config := `repos:
  - id: repora
    canonical:
      provider: gitlab
      path: micrantha/repora
    mirrors:
      - provider: github
        path: hackelia-micrantha/repora
    artifacts:
      readme:
        template: "` + strings.ReplaceAll(template, `\`, `\\`) + `"
`
			writeFile(t, path, []byte(config))
			_, err := Load(path)
			if err == nil {
				t.Fatalf("Load accepted unsafe template path %q", template)
			}
		})
	}
}

func TestLoadRejectsInvalidREADMEValueKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repora.yaml")
	writeFile(t, path, []byte(`repos:
  - id: repora
    canonical:
      provider: gitlab
      path: micrantha/repora
    mirrors:
      - provider: github
        path: hackelia-micrantha/repora
    artifacts:
      readme:
        template: templates/README.md.tmpl
        values:
          bad.key: nope
`))

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "invalid artifacts.readme.values key") {
		t.Fatalf("Load error = %v, want invalid value-key rejection", err)
	}
}
