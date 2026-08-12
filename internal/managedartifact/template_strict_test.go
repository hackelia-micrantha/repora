package managedartifact

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTemplateRejectsNonCanonicalReferencesBeforeResolution(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "repora.yaml")
	if err := os.WriteFile(configPath, []byte("repos: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, reference := range []string{
		" templates/README.md.tmpl",
		"templates/README.md.tmpl ",
		"templates/../README.md.tmpl",
		"./README.md.tmpl",
		"https://example.test/README.md.tmpl",
		"~/README.md.tmpl",
		"templates/repo\u202e.tmpl",
	} {
		t.Run(reference, func(t *testing.T) {
			if _, err := LoadTemplate(configPath, reference); err == nil {
				t.Fatalf("LoadTemplate(%q) returned nil error", reference)
			}
		})
	}
}
