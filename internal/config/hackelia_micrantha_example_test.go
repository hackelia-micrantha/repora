package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadHackeliaMicranthaExample(t *testing.T) {
	t.Parallel()

	spec, err := Load(filepath.Join("..", "..", "examples", "hackelia-micrantha", "repora.yaml"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(spec.Repos) != 62 {
		t.Fatalf("repo count = %d, want 62", len(spec.Repos))
	}

	for _, repo := range spec.Repos {
		if repo.Canonical.Provider != "gitlab" || !strings.HasPrefix(repo.Canonical.Path, "replace-with-gitlab-namespace/") {
			t.Errorf("%q canonical = %#v", repo.ID, repo.Canonical)
		}
		if len(repo.Mirrors) != 1 {
			t.Errorf("%q mirror count = %d, want 1", repo.ID, len(repo.Mirrors))
			continue
		}
		mirror := repo.Mirrors[0]
		wantPath := "hackelia-micrantha/" + repo.ID
		if mirror.Provider != "github" || mirror.Path != wantPath {
			t.Errorf("%q mirror = %#v, want github:%s", repo.ID, mirror, wantPath)
		}
	}
}
