package managedartifact

import (
	"strings"
	"testing"

	"repoctl/internal/config"
)

func TestBuildPlanRequiresObserverOnlyForConfiguredArtifacts(t *testing.T) {
	spec := configuredPlannerSpec("repo.repora", "repora", "templates/README.md.tmpl", nil)
	_, err := BuildPlan("unused.yaml", spec, nil)
	if err == nil || !strings.Contains(err.Error(), "observer is required") {
		t.Fatalf("BuildPlan() error = %v, want configured-observer requirement", err)
	}
}

func TestBuildPlanRejectsNonCanonicalIdentityBeforeTemplateIO(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*config.Repo)
	}{
		{name: "uid", mutate: func(repo *config.Repo) { repo.UID = "repo bad" }},
		{name: "provider", mutate: func(repo *config.Repo) { repo.Canonical.Provider = "gitlab " }},
		{name: "path trailing slash", mutate: func(repo *config.Repo) { repo.Canonical.Path += "/" }},
		{name: "path display control", mutate: func(repo *config.Repo) { repo.Canonical.Path = "micrantha/repo\u202e" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := configuredPlannerSpec("repo.repora", "repora", "templates/does-not-exist.tmpl", nil)
			tc.mutate(&spec.Repos[0])
			observer := &fakeREADMEObserver{}
			_, err := BuildPlan("does-not-exist.yaml", spec, observer)
			if err == nil || strings.Contains(err.Error(), "template") || strings.Contains(err.Error(), "configuration path") {
				t.Fatalf("BuildPlan() error = %v, want identity rejection before template I/O", err)
			}
			if len(observer.calls) != 0 {
				t.Fatalf("observer calls = %v, want none before identity validation", observer.calls)
			}
		})
	}
}
