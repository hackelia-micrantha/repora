package managedartifact

import (
	"path/filepath"
	"testing"

	"repoctl/internal/config"
)

func TestManagedREADMEExampleConfigAndTemplate(t *testing.T) {
	configPath := filepath.Join("..", "..", "examples", "managed-readme", "repora.yaml")
	spec, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load managed README example config: %v", err)
	}
	if len(spec.Repos) != 1 || spec.Repos[0].Artifacts.Readme == nil {
		t.Fatalf("example config = %+v, want one managed README repo", spec)
	}

	repo := spec.Repos[0]
	template, err := LoadTemplate(configPath, repo.Artifacts.Readme.Template)
	if err != nil {
		t.Fatalf("load managed README example template: %v", err)
	}
	rendered, err := RenderREADME(template, RenderData{
		RepoID:            repo.ID,
		RepoUID:           repo.DurableID(),
		CanonicalProvider: repo.Canonical.Provider,
		CanonicalPath:     repo.Canonical.Path,
		Values:            repo.Artifacts.Readme.Values,
	})
	if err != nil {
		t.Fatalf("render managed README example: %v", err)
	}

	want := "# demo\n\nA deterministic README managed by Repora.\n\n- UID: `repo.demo`\n- Canonical provider: `gitlab`\n- Canonical path: `example/demo`\n"
	if string(rendered) != want {
		t.Fatalf("rendered example = %q, want %q", rendered, want)
	}
}
