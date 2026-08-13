package managedartifact

import (
	"errors"
	"strings"
	"testing"

	"repoctl/internal/config"
)

type preflightObserver struct {
	observation READMEObservation
	err         error
	calls       int
}

func (o *preflightObserver) ObserveREADME(config.Repo) (READMEObservation, error) {
	o.calls++
	return o.observation, o.err
}

func TestPreflightPlanAcceptsExactReviewedState(t *testing.T) {
	spec, plan, observation := preflightFixture(t)
	observer := &preflightObserver{observation: observation}
	if err := PreflightPlan(spec, plan, observer); err != nil {
		t.Fatal(err)
	}
	if observer.calls != 1 {
		t.Fatalf("observer calls = %d, want 1", observer.calls)
	}
}

func TestPreflightPlanEmptyRequiresNoObserver(t *testing.T) {
	plan := Plan{Kind: PlanKind, Version: PlanVersion, Repositories: []RepositoryPlan{}}
	if err := PreflightPlan(config.Spec{}, plan, nil); err != nil {
		t.Fatal(err)
	}
}

func TestPreflightPlanRejectsConfigurationDriftBeforeObservation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*config.Spec)
		want   string
	}{
		{name: "uid removed", mutate: func(spec *config.Spec) { spec.Repos = nil }, want: "no longer configured"},
		{name: "authority removed", mutate: func(spec *config.Spec) { spec.Repos[0].Artifacts.Readme = nil }, want: "no longer enables managed README authority"},
		{name: "id changed", mutate: func(spec *config.Spec) { spec.Repos[0].ID = "renamed" }, want: "id changed"},
		{name: "provider changed", mutate: func(spec *config.Spec) { spec.Repos[0].Canonical.Provider = "github" }, want: "canonical target changed"},
		{name: "path changed", mutate: func(spec *config.Spec) { spec.Repos[0].Canonical.Path = "example/other" }, want: "canonical target changed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec, plan, observation := preflightFixture(t)
			tc.mutate(&spec)
			observer := &preflightObserver{observation: observation}
			err := PreflightPlan(spec, plan, observer)
			if !errors.Is(err, ErrStale) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want stale containing %q", err, tc.want)
			}
			if observer.calls != 0 {
				t.Fatalf("observer calls = %d, want 0 before config preflight succeeds", observer.calls)
			}
		})
	}
}

func TestPreflightPlanRejectsCanonicalStateDrift(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*READMEObservation)
		want   string
	}{
		{name: "branch", mutate: func(o *READMEObservation) { o.Branch = "trunk" }, want: "default branch changed"},
		{name: "head", mutate: func(o *READMEObservation) { o.BaseOID = strings.Repeat("2", 40) }, want: "canonical HEAD changed"},
		{name: "presence", mutate: func(o *READMEObservation) { o.Present = false; o.Mode = ""; o.Content = nil }, want: "README presence changed"},
		{name: "mode", mutate: func(o *READMEObservation) { o.Mode = "100755" }, want: "README mode changed"},
		{name: "content", mutate: func(o *READMEObservation) { o.Content = []byte("changed\n") }, want: "content digest changed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec, plan, observation := preflightFixture(t)
			tc.mutate(&observation)
			err := PreflightPlan(spec, plan, &preflightObserver{observation: observation})
			if !errors.Is(err, ErrStale) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want stale containing %q", err, tc.want)
			}
		})
	}
}

func TestPreflightPlanRejectsReviewedDiffMismatch(t *testing.T) {
	spec, plan, observation := preflightFixture(t)
	plan.Repositories[0].Actions[0].Diff = "--- a/README.md\n+++ b/README.md\n@@ tampered @@\n-\"old\\n\"\n+\"new\\n\"\n"
	if err := plan.Validate(); err != nil {
		t.Fatalf("tampered but structurally valid plan: %v", err)
	}
	err := PreflightPlan(spec, plan, &preflightObserver{observation: observation})
	if !errors.Is(err, ErrStale) || !strings.Contains(err.Error(), "reviewed README diff") {
		t.Fatalf("error = %v, want reviewed-diff stale error", err)
	}
}

func TestPreflightPlanObserverFailureIsNotStale(t *testing.T) {
	spec, plan, observation := preflightFixture(t)
	err := PreflightPlan(spec, plan, &preflightObserver{observation: observation, err: errors.New("fetch unavailable")})
	if err == nil || errors.Is(err, ErrStale) || !strings.Contains(err.Error(), "fetch unavailable") {
		t.Fatalf("error = %v, want non-stale observer failure", err)
	}
}

func preflightFixture(t *testing.T) (config.Spec, Plan, READMEObservation) {
	t.Helper()
	observed := []byte("old\n")
	desired := "new\n"
	diff, err := ReviewDiff(true, observed, []byte(desired))
	if err != nil {
		t.Fatal(err)
	}
	present := true
	spec := config.Spec{Repos: []config.Repo{{
		ID:  "demo",
		UID: "repo.demo",
		Canonical: config.Endpoint{
			Provider: "gitlab",
			Path:     "example/demo",
		},
		Mirrors: []config.Endpoint{{Provider: "github", Path: "example/demo"}},
		Artifacts: config.RepositoryArtifacts{Readme: &config.ReadmeArtifact{
			Template: "templates/README.md.tmpl",
		}},
	}}}
	plan := Plan{
		Kind:    PlanKind,
		Version: PlanVersion,
		Repositories: []RepositoryPlan{{
			UID: "repo.demo",
			ID:  "demo",
			Target: Target{
				Provider: "gitlab",
				Path:     "example/demo",
				Branch:   "main",
			},
			BaseOID: strings.Repeat("1", 40),
			Actions: []Action{{
				Type: ActionWriteREADME,
				Path: READMEPath,
				Observed: ObservedState{
					Present: &present,
					Mode:    "100644",
					SHA256:  DigestSHA256(observed),
				},
				Desired: DesiredState{
					Mode:    "100644",
					SHA256:  DigestSHA256([]byte(desired)),
					Content: &desired,
				},
				TemplateSHA256: strings.Repeat("c", 64),
				Diff:           diff,
			}},
		}},
	}
	observation := READMEObservation{
		Branch:  "main",
		BaseOID: strings.Repeat("1", 40),
		Present: true,
		Mode:    "100644",
		Content: append([]byte(nil), observed...),
	}
	return spec, plan, observation
}
