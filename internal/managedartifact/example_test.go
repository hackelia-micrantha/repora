package managedartifact_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/managedartifact"
)

func TestManagedReadmeExampleRendersDeterministically(t *testing.T) {
	configPath := filepath.Join("..", "..", "examples", "managed-readme", "repora.yaml")
	spec, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load example config: %v", err)
	}
	if len(spec.Repos) != 1 {
		t.Fatalf("example repo count = %d, want 1", len(spec.Repos))
	}

	repo := spec.Repos[0]
	if repo.Artifacts.Readme == nil {
		t.Fatal("example README artifact is not configured")
	}
	template, err := managedartifact.LoadTemplate(configPath, repo.Artifacts.Readme.Template)
	if err != nil {
		t.Fatalf("load example template: %v", err)
	}

	data := managedartifact.RenderData{
		RepoID:            repo.ID,
		RepoUID:           repo.UID,
		CanonicalProvider: repo.Canonical.Provider,
		CanonicalPath:     repo.Canonical.Path,
		Values:            repo.Artifacts.Readme.Values,
	}
	first, err := managedartifact.RenderREADME(template, data)
	if err != nil {
		t.Fatalf("render example README: %v", err)
	}
	second, err := managedartifact.RenderREADME(template, data)
	if err != nil {
		t.Fatalf("render example README again: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("example README rendering is not deterministic")
	}

	const want = "# Anthesis\n\nGoverned automation with explicit evidence and policy boundaries.\n\nRepository: `anthesis` (`repo.anthesis`)\nCanonical: `gitlab:micrantha/anthesis`\n"
	if got := string(first); got != want {
		t.Fatalf("rendered example README mismatch:\n--- got ---\n%s--- want ---\n%s", got, want)
	}
}
