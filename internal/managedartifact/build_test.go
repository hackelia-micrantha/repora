package managedartifact

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"repoctl/internal/config"
)

type fakeREADMEObserver struct {
	byUID map[string]READMEObservation
	err   error
	calls []string
}

func (observer *fakeREADMEObserver) ObserveREADME(repo config.Repo) (READMEObservation, error) {
	observer.calls = append(observer.calls, repo.DurableID())
	if observer.err != nil {
		return READMEObservation{}, observer.err
	}
	value, ok := observer.byUID[repo.DurableID()]
	if !ok {
		return READMEObservation{}, errors.New("missing fake observation")
	}
	return value, nil
}

func TestBuildPlanLeavesUnconfiguredSpecUntouchedWithoutObserver(t *testing.T) {
	plan, err := BuildPlan("does-not-need-to-exist.yaml", config.Spec{Repos: []config.Repo{{ID: "plain"}}}, nil)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if plan.Repositories == nil || len(plan.Repositories) != 0 {
		t.Fatalf("BuildPlan() repositories = %#v, want explicit empty array", plan.Repositories)
	}
}

func TestBuildPlanBuildsExactREADMEActionAndPreservesMode(t *testing.T) {
	configPath := writePlannerConfigAndTemplate(t, "# {{value.title}}\n")
	spec := configuredPlannerSpec("repo.repora", "repora", "templates/README.md.tmpl", map[string]string{"title": "New"})
	observer := &fakeREADMEObserver{byUID: map[string]READMEObservation{
		"repo.repora": {
			Branch:  "main",
			BaseOID: strings.Repeat("a", 40),
			Present: true,
			Mode:    "100755",
			Content: []byte("# Old\r\n"),
		},
	}}

	plan, err := BuildPlan(configPath, spec, observer)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if len(plan.Repositories) != 1 {
		t.Fatalf("repositories = %d, want 1", len(plan.Repositories))
	}
	repoPlan := plan.Repositories[0]
	if repoPlan.UID != "repo.repora" || repoPlan.Target.Provider != "gitlab" || repoPlan.Target.Path != "micrantha/repora" || repoPlan.Target.Branch != "main" {
		t.Fatalf("repository identity = %#v", repoPlan)
	}
	action := repoPlan.Actions[0]
	if action.Observed.Mode != "100755" || action.Desired.Mode != "100755" {
		t.Fatalf("modes observed=%q desired=%q", action.Observed.Mode, action.Desired.Mode)
	}
	if action.Observed.SHA256 != DigestSHA256([]byte("# Old\r\n")) {
		t.Fatalf("observed digest = %q", action.Observed.SHA256)
	}
	if action.Desired.Content == nil || *action.Desired.Content != "# New\n" || action.Desired.SHA256 != DigestSHA256([]byte("# New\n")) {
		t.Fatalf("desired = %#v", action.Desired)
	}
	if action.TemplateSHA256 != DigestSHA256([]byte("# {{value.title}}\n")) {
		t.Fatalf("template digest = %q", action.TemplateSHA256)
	}
	if !strings.Contains(action.Diff, `-"# Old\r\n"`) || !strings.Contains(action.Diff, `+"# New\n"`) {
		t.Fatalf("diff does not expose exact line endings:\n%s", action.Diff)
	}
}

func TestBuildPlanOmitsAlreadyEqualREADME(t *testing.T) {
	configPath := writePlannerConfigAndTemplate(t, "# Same\n")
	spec := configuredPlannerSpec("repo.repora", "repora", "templates/README.md.tmpl", nil)
	observer := &fakeREADMEObserver{byUID: map[string]READMEObservation{
		"repo.repora": {
			Branch:  "main",
			BaseOID: strings.Repeat("b", 40),
			Present: true,
			Mode:    "100644",
			Content: []byte("# Same\n"),
		},
	}}

	plan, err := BuildPlan(configPath, spec, observer)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if len(plan.Repositories) != 0 {
		t.Fatalf("repositories = %#v, want no-op omitted", plan.Repositories)
	}
}

func TestBuildPlanRepresentsCreationOfEmptyREADME(t *testing.T) {
	configPath := writePlannerConfigAndTemplate(t, "")
	spec := configuredPlannerSpec("repo.repora", "repora", "templates/README.md.tmpl", nil)
	observer := &fakeREADMEObserver{byUID: map[string]READMEObservation{
		"repo.repora": {
			Branch:  "main",
			BaseOID: strings.Repeat("c", 40),
			Present: false,
		},
	}}

	plan, err := BuildPlan(configPath, spec, observer)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	action := plan.Repositories[0].Actions[0]
	if action.Observed.Present == nil || *action.Observed.Present {
		t.Fatalf("observed present = %#v", action.Observed.Present)
	}
	if action.Desired.Mode != "100644" || action.Desired.Content == nil || *action.Desired.Content != "" {
		t.Fatalf("desired = %#v", action.Desired)
	}
	if !strings.Contains(action.Diff, `+""`) {
		t.Fatalf("diff = %q, want explicit empty-file creation", action.Diff)
	}
}

func TestBuildPlanSortsConfiguredRepositoriesByDurableID(t *testing.T) {
	configPath := writePlannerConfigAndTemplate(t, "# New\n")
	first := configuredPlannerSpec("repo.z", "z", "templates/README.md.tmpl", nil).Repos[0]
	second := configuredPlannerSpec("repo.a", "a", "templates/README.md.tmpl", nil).Repos[0]
	spec := config.Spec{Repos: []config.Repo{first, second}}
	observer := &fakeREADMEObserver{byUID: map[string]READMEObservation{
		"repo.a": {Branch: "main", BaseOID: strings.Repeat("a", 40), Present: false},
		"repo.z": {Branch: "main", BaseOID: strings.Repeat("b", 40), Present: false},
	}}

	plan, err := BuildPlan(configPath, spec, observer)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if len(plan.Repositories) != 2 || plan.Repositories[0].UID != "repo.a" || plan.Repositories[1].UID != "repo.z" {
		t.Fatalf("repository order = %#v", plan.Repositories)
	}
	if got := strings.Join(observer.calls, ","); got != "repo.a,repo.z" {
		t.Fatalf("observer order = %q", got)
	}
}

func TestBuildPlanRejectsUnsafeObservationBeforePlanEmission(t *testing.T) {
	configPath := writePlannerConfigAndTemplate(t, "# New\n")
	spec := configuredPlannerSpec("repo.repora", "repora", "templates/README.md.tmpl", nil)
	observer := &fakeREADMEObserver{byUID: map[string]READMEObservation{
		"repo.repora": {
			Branch:  "main",
			BaseOID: strings.Repeat("a", 40),
			Present: true,
			Mode:    "120000",
			Content: []byte("target"),
		},
	}}
	_, err := BuildPlan(configPath, spec, observer)
	if err == nil || !strings.Contains(err.Error(), "regular Git blob") {
		t.Fatalf("BuildPlan() error = %v, want non-regular-mode rejection", err)
	}
}

func configuredPlannerSpec(uid, id, template string, values map[string]string) config.Spec {
	return config.Spec{Repos: []config.Repo{
		{
			ID:  id,
			UID: uid,
			Canonical: config.Endpoint{
				Provider: "gitlab",
				Path:     "micrantha/" + id,
			},
			Mirrors: []config.Endpoint{{Provider: "github", Path: "hackelia-micrantha/" + id}},
			Mode:    "mirror",
			Artifacts: config.RepositoryArtifacts{Readme: &config.ReadmeArtifact{
				Template: template,
				Values:   values,
			}},
		},
	}}
}

func writePlannerConfigAndTemplate(t *testing.T, template string) string {
	t.Helper()
	root := t.TempDir()
	configPath := filepath.Join(root, "repora.yaml")
	if err := os.WriteFile(configPath, []byte("repos: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "templates"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "templates", "README.md.tmpl"), []byte(template), 0o600); err != nil {
		t.Fatal(err)
	}
	return configPath
}
