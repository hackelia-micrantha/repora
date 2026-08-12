package managedartifact

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestPlanMarshalParseRoundTripDeterministically(t *testing.T) {
	plan := validManagedPlan()
	first, err := plan.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	second, err := plan.Marshal()
	if err != nil {
		t.Fatalf("Marshal() second error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("Marshal() is not deterministic\nfirst: %s\nsecond: %s", first, second)
	}

	parsed, err := ParsePlan(first)
	if err != nil {
		t.Fatalf("ParsePlan() error = %v", err)
	}
	if !reflect.DeepEqual(parsed, plan) {
		t.Fatalf("ParsePlan(Marshal(plan)) mismatch\ngot: %#v\nwant: %#v", parsed, plan)
	}
}

func TestPlanAllowsExplicitEmptyRepositorySet(t *testing.T) {
	plan := Plan{Kind: PlanKind, Version: PlanVersion, Repositories: []RepositoryPlan{}}
	data, err := plan.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if _, err := ParsePlan(data); err != nil {
		t.Fatalf("ParsePlan() error = %v", err)
	}
}

func TestParsePlanRejectsMissingRepositoriesArray(t *testing.T) {
	_, err := ParsePlan([]byte(`{"kind":"repora.io/managed-artifact-plan","version":1}`))
	if err == nil || !strings.Contains(err.Error(), "repositories array is required") {
		t.Fatalf("ParsePlan() error = %v", err)
	}
}

func TestParsePlanRejectsUnknownFieldNullAndTrailingData(t *testing.T) {
	data, err := validManagedPlan().Marshal()
	if err != nil {
		t.Fatal(err)
	}

	unknown := bytes.Replace(data, []byte(`"version": 1,`), []byte(`"version": 1, "unexpected": true,`), 1)
	if _, err := ParsePlan(unknown); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown-field ParsePlan() error = %v", err)
	}

	nullValue := bytes.Replace(data, []byte(`"diff": "`), []byte(`"diff": null, "removed_diff": "`), 1)
	if _, err := ParsePlan(nullValue); err == nil || !strings.Contains(err.Error(), "must not be null") {
		t.Fatalf("null ParsePlan() error = %v", err)
	}

	trailing := append(append([]byte(nil), data...), []byte(` {}`)...)
	if _, err := ParsePlan(trailing); err == nil || !strings.Contains(err.Error(), "trailing JSON value") {
		t.Fatalf("trailing ParsePlan() error = %v", err)
	}
}

func TestPlanRejectsDuplicateRepositoryIdentityAndTarget(t *testing.T) {
	base := validManagedPlan().Repositories[0]

	duplicateUID := validManagedPlan()
	second := base
	second.ID = "repora-two"
	second.Target.Path = "micrantha/repora-two"
	duplicateUID.Repositories = append(duplicateUID.Repositories, second)
	if err := duplicateUID.Validate(); err == nil || !strings.Contains(err.Error(), "duplicate managed artifact repository uid") {
		t.Fatalf("duplicate UID Validate() error = %v", err)
	}

	duplicateID := validManagedPlan()
	second = base
	second.UID = "repo.repora-two"
	second.Target.Path = "micrantha/repora-two"
	duplicateID.Repositories = append(duplicateID.Repositories, second)
	if err := duplicateID.Validate(); err == nil || !strings.Contains(err.Error(), "duplicate managed artifact repository id") {
		t.Fatalf("duplicate ID Validate() error = %v", err)
	}

	duplicateTarget := validManagedPlan()
	second = base
	second.UID = "repo.repora-two"
	second.ID = "repora-two"
	duplicateTarget.Repositories = append(duplicateTarget.Repositories, second)
	if err := duplicateTarget.Validate(); err == nil || !strings.Contains(err.Error(), "duplicate managed artifact target") {
		t.Fatalf("duplicate target Validate() error = %v", err)
	}
}

func TestPlanValidatesObservedAndDesiredREADMEState(t *testing.T) {
	t.Run("absent README creates regular file", func(t *testing.T) {
		plan := validManagedPlan()
		action := &plan.Repositories[0].Actions[0]
		action.Observed = ObservedState{Present: boolPointer(false)}
		action.Desired.Mode = "100644"
		action.Diff = "--- a/README.md\n+++ b/README.md\n@@ -0,0 +1,1 @@\n+# New\n"
		if err := plan.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("absent README rejects observed mode", func(t *testing.T) {
		plan := validManagedPlan()
		action := &plan.Repositories[0].Actions[0]
		action.Observed = ObservedState{Present: boolPointer(false), Mode: "100644"}
		if err := plan.Validate(); err == nil || !strings.Contains(err.Error(), "must not define mode or sha256") {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("new README rejects executable mode", func(t *testing.T) {
		plan := validManagedPlan()
		action := &plan.Repositories[0].Actions[0]
		action.Observed = ObservedState{Present: boolPointer(false)}
		action.Desired.Mode = "100755"
		if err := plan.Validate(); err == nil || !strings.Contains(err.Error(), "new README must use mode 100644") {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("existing README mode is preserved", func(t *testing.T) {
		plan := validManagedPlan()
		plan.Repositories[0].Actions[0].Desired.Mode = "100755"
		if err := plan.Validate(); err == nil || !strings.Contains(err.Error(), "preserve observed regular-file mode") {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("desired digest matches exact bytes", func(t *testing.T) {
		plan := validManagedPlan()
		plan.Repositories[0].Actions[0].Desired.SHA256 = strings.Repeat("0", 64)
		if err := plan.Validate(); err == nil || !strings.Contains(err.Error(), "does not match content") {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("empty desired content remains representable", func(t *testing.T) {
		plan := validManagedPlan()
		action := &plan.Repositories[0].Actions[0]
		empty := ""
		action.Desired.Content = &empty
		action.Desired.SHA256 = DigestSHA256(nil)
		action.Diff = "--- a/README.md\n+++ b/README.md\n@@ -1,1 +0,0 @@\n-# Old\n"
		if err := plan.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("missing desired content is rejected", func(t *testing.T) {
		plan := validManagedPlan()
		plan.Repositories[0].Actions[0].Desired.Content = nil
		if err := plan.Validate(); err == nil || !strings.Contains(err.Error(), "content field is required") {
			t.Fatalf("Validate() error = %v", err)
		}
	})
}

func TestPlanRejectsNoOpREADMEAction(t *testing.T) {
	plan := validManagedPlan()
	action := &plan.Repositories[0].Actions[0]
	content := "# Old\n"
	action.Desired.Content = &content
	action.Desired.SHA256 = action.Observed.SHA256
	if err := plan.Validate(); err == nil || !strings.Contains(err.Error(), "must not contain a no-op") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestPlanRejectsUnsafeIdentityAndReviewDiff(t *testing.T) {
	for name, mutate := range map[string]func(*Plan){
		"uid whitespace":      func(plan *Plan) { plan.Repositories[0].UID = " repo.repora" },
		"provider whitespace": func(plan *Plan) { plan.Repositories[0].Target.Provider = "gitlab " },
		"path whitespace":     func(plan *Plan) { plan.Repositories[0].Target.Path = " micrantha/repora" },
		"branch whitespace":   func(plan *Plan) { plan.Repositories[0].Target.Branch = "main " },
	} {
		t.Run(name, func(t *testing.T) {
			plan := validManagedPlan()
			mutate(&plan)
			if err := plan.Validate(); err == nil {
				t.Fatal("Validate() accepted non-canonical whitespace")
			}
		})
	}

	t.Run("provider path", func(t *testing.T) {
		plan := validManagedPlan()
		plan.Repositories[0].Target.Path = "micrantha/repo name"
		if err := plan.Validate(); err == nil || !strings.Contains(err.Error(), "unsafe segment") {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("branch", func(t *testing.T) {
		plan := validManagedPlan()
		plan.Repositories[0].Target.Branch = "../main"
		if err := plan.Validate(); err == nil || !strings.Contains(err.Error(), "valid symbolic ref") {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("fixed README labels", func(t *testing.T) {
		plan := validManagedPlan()
		plan.Repositories[0].Actions[0].Diff = "--- a/OTHER.md\n+++ b/OTHER.md\n@@ -1 +1 @@\n-old\n+new\n"
		if err := plan.Validate(); err == nil || !strings.Contains(err.Error(), "fixed README.md labels") {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("terminal controls", func(t *testing.T) {
		plan := validManagedPlan()
		plan.Repositories[0].Actions[0].Diff = "--- a/README.md\n+++ b/README.md\n@@ -1 +1 @@\n-old\n+\x1b[31mnew\n"
		if err := plan.Validate(); err == nil || !strings.Contains(err.Error(), "control character") {
			t.Fatalf("Validate() error = %v", err)
		}
	})
}

func TestPlanRejectsUnsafeRenderedContent(t *testing.T) {
	plan := validManagedPlan()
	content := "safe\x1b[31munsafe\n"
	plan.Repositories[0].Actions[0].Desired.Content = &content
	plan.Repositories[0].Actions[0].Desired.SHA256 = DigestSHA256([]byte(content))
	if err := plan.Validate(); err == nil || !strings.Contains(err.Error(), "control character") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func validManagedPlan() Plan {
	oldContent := "# Old\n"
	newContent := "# New\n"
	return Plan{
		Kind:    PlanKind,
		Version: PlanVersion,
		Repositories: []RepositoryPlan{
			{
				UID: "repo.repora",
				ID:  "repora",
				Target: Target{
					Provider: "gitlab",
					Path:     "micrantha/repora",
					Branch:   "main",
				},
				BaseOID: strings.Repeat("a", 40),
				Actions: []Action{
					{
						Type: ActionWriteREADME,
						Path: READMEPath,
						Observed: ObservedState{
							Present: boolPointer(true),
							Mode:    "100644",
							SHA256:  DigestSHA256([]byte(oldContent)),
						},
						Desired: DesiredState{
							Mode:    "100644",
							SHA256:  DigestSHA256([]byte(newContent)),
							Content: stringPointer(newContent),
						},
						TemplateSHA256: DigestSHA256([]byte("# {{value.title}}\n")),
						Diff:           "--- a/README.md\n+++ b/README.md\n@@ -1,1 +1,1 @@\n-# Old\n+# New\n",
					},
				},
			},
		},
	}
}

func boolPointer(value bool) *bool {
	return &value
}

func stringPointer(value string) *string {
	return &value
}
