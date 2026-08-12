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
		{name: "legacy canonical URL", mutate: func(repo *config.Repo) { repo.Canonical.URL = "https://example.test/repo.git" }},
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

func TestBuildPlanRejectsDuplicateIdentityBeforeIO(t *testing.T) {
	first := configuredPlannerSpec("repo.same", "one", "templates/does-not-exist.tmpl", nil).Repos[0]
	second := configuredPlannerSpec("repo.same", "two", "templates/does-not-exist.tmpl", nil).Repos[0]
	observer := &fakeREADMEObserver{}
	_, err := BuildPlan("does-not-exist.yaml", config.Spec{Repos: []config.Repo{first, second}}, observer)
	if err == nil || !strings.Contains(err.Error(), "duplicate managed artifact repository uid") {
		t.Fatalf("BuildPlan() error = %v, want duplicate UID rejection", err)
	}
	if len(observer.calls) != 0 {
		t.Fatalf("observer calls = %v, want none before duplicate rejection", observer.calls)
	}
}

func TestBuildPlanRejectsUnsafeObservedBranch(t *testing.T) {
	configPath := writePlannerConfigAndTemplate(t, "# New\n")
	spec := configuredPlannerSpec("repo.repora", "repora", "templates/README.md.tmpl", nil)
	observer := &fakeREADMEObserver{byUID: map[string]READMEObservation{
		"repo.repora": {Branch: " main", BaseOID: strings.Repeat("a", 40), Present: false},
	}}
	_, err := BuildPlan(configPath, spec, observer)
	if err == nil || !strings.Contains(err.Error(), "valid symbolic ref") {
		t.Fatalf("BuildPlan() error = %v, want branch rejection", err)
	}
}
