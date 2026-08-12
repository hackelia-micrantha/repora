package managedartifact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadTemplateReadsContainedRegularFile(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "repora.yaml")
	if err := os.WriteFile(configPath, []byte("repos: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "templates"), 0o700); err != nil {
		t.Fatal(err)
	}
	want := []byte("# {{repo.id}}\r\n")
	if err := os.WriteFile(filepath.Join(root, "templates", "README.md.tmpl"), want, 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := LoadTemplate(configPath, "templates/README.md.tmpl")
	if err != nil {
		t.Fatalf("LoadTemplate() error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("LoadTemplate() = %q, want exact bytes %q", got, want)
	}
}

func TestLoadTemplateResolvesConfigSymlinkRelativeToPhysicalConfig(t *testing.T) {
	physical := t.TempDir()
	linkRoot := t.TempDir()
	configPath := filepath.Join(physical, "repora.yaml")
	if err := os.WriteFile(configPath, []byte("repos: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(physical, "templates"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(physical, "templates", "README.md.tmpl"), []byte("physical"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(linkRoot, "repora.yaml")
	if err := os.Symlink(configPath, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	got, err := LoadTemplate(link, "templates/README.md.tmpl")
	if err != nil {
		t.Fatalf("LoadTemplate() error = %v", err)
	}
	if string(got) != "physical" {
		t.Fatalf("LoadTemplate() = %q", got)
	}
}

func TestLoadTemplateRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	configPath := filepath.Join(root, "repora.yaml")
	if err := os.WriteFile(configPath, []byte("repos: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	outsideTemplate := filepath.Join(outside, "README.md.tmpl")
	if err := os.WriteFile(outsideTemplate, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "README.md.tmpl")
	if err := os.Symlink(outsideTemplate, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	_, err := LoadTemplate(configPath, "README.md.tmpl")
	if err == nil || !strings.Contains(err.Error(), "outside configuration root") {
		t.Fatalf("LoadTemplate() error = %v, want containment rejection", err)
	}
}

func TestLoadTemplateRejectsTraversalOversizeAndNonFileConfig(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "repora.yaml")
	if err := os.WriteFile(configPath, []byte("repos: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadTemplate(configPath, "../README.md.tmpl"); err == nil {
		t.Fatal("LoadTemplate accepted traversal")
	}

	largePath := filepath.Join(root, "large.tmpl")
	file, err := os.Create(largePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(MaxTextBytes + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadTemplate(configPath, "large.tmpl"); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("LoadTemplate() oversize error = %v", err)
	}

	if _, err := LoadTemplate(root, "README.md.tmpl"); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("LoadTemplate(directory config) error = %v, want regular-file rejection", err)
	}
}
