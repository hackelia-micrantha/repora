package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/managedartifact"
)

func TestPlanREADMEHumanReview(t *testing.T) {
	configPath := writeManagedPlanConfig(t)
	plan := managedPlanFixture(t)
	stubManagedPlanBuild(t, plan, nil)

	var stdout bytes.Buffer
	code := captureManagedStdout(t, &stdout, func() int {
		return run([]string{"plan-readme", "-f", configPath})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	got := stdout.String()
	for _, want := range []string{
		"demo (repo.demo) gitlab/example/demo#main\n",
		"base: 1111111111111111111111111111111111111111\n",
		"README.md: 100644 " + strings.Repeat("a", 64) + " -> 100644 " + strings.Repeat("b", 64) + "\n",
		"--- a/README.md\n+++ b/README.md\n@@ review @@\n-\"old\\n\"\n+\"new\\n\"\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, missing %q", got, want)
		}
	}
}

func TestPlanREADMEArtifactOutputParsesAsManagedPlan(t *testing.T) {
	configPath := writeManagedPlanConfig(t)
	plan := managedPlanFixture(t)
	stubManagedPlanBuild(t, plan, nil)

	var stdout bytes.Buffer
	code := captureManagedStdout(t, &stdout, func() int {
		return run([]string{"plan-readme", "-f", configPath, "--artifact"})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	parsed, err := managedartifact.ParsePlan(stdout.Bytes())
	if err != nil {
		t.Fatalf("parse emitted managed plan: %v\n%s", err, stdout.String())
	}
	if parsed.Repositories[0].UID != "repo.demo" {
		t.Fatalf("uid = %q, want repo.demo", parsed.Repositories[0].UID)
	}
}

func TestPlanREADMENoChanges(t *testing.T) {
	configPath := writeManagedPlanConfig(t)
	stubManagedPlanBuild(t, managedartifact.Plan{Kind: managedartifact.PlanKind, Version: managedartifact.PlanVersion, Repositories: []managedartifact.RepositoryPlan{}}, nil)

	var stdout bytes.Buffer
	code := captureManagedStdout(t, &stdout, func() int {
		return run([]string{"plan-readme", "-f", configPath})
	})
	if code != 0 || stdout.String() != "No managed README changes.\n" {
		t.Fatalf("code=%d stdout=%q", code, stdout.String())
	}
}

func TestPlanREADMERejectsPositionals(t *testing.T) {
	configPath := writeManagedPlanConfig(t)
	var stderr bytes.Buffer
	code := captureManagedStderr(t, &stderr, func() int {
		return run([]string{"plan-readme", "-f", configPath, "extra"})
	})
	if code != 1 || !strings.Contains(stderr.String(), "does not accept positional arguments") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}

func TestPlanREADMEPropagatesPlannerFailureWithoutOutput(t *testing.T) {
	configPath := writeManagedPlanConfig(t)
	stubManagedPlanBuild(t, managedartifact.Plan{}, errors.New("observe failed"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := captureManagedStdout(t, &stdout, func() int {
		return captureManagedStderr(t, &stderr, func() int {
			return run([]string{"plan-readme", "-f", configPath})
		})
	})
	if code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "observe failed") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func stubManagedPlanBuild(t *testing.T, plan managedartifact.Plan, buildErr error) {
	t.Helper()
	oldBuild := managedREADMEPlanBuild
	oldObserver := managedREADMEObserver
	managedREADMEPlanBuild = func(string, config.Spec, managedartifact.READMEObserver) (managedartifact.Plan, error) {
		return plan, buildErr
	}
	managedREADMEObserver = func() managedartifact.READMEObserver { return nil }
	t.Cleanup(func() {
		managedREADMEPlanBuild = oldBuild
		managedREADMEObserver = oldObserver
	})
}

func managedPlanFixture(t *testing.T) managedartifact.Plan {
	t.Helper()
	present := true
	desired := "new\n"
	plan := managedartifact.Plan{
		Kind:    managedartifact.PlanKind,
		Version: managedartifact.PlanVersion,
		Repositories: []managedartifact.RepositoryPlan{{
			UID: "repo.demo",
			ID:  "demo",
			Target: managedartifact.Target{
				Provider: "gitlab",
				Path:     "example/demo",
				Branch:   "main",
			},
			BaseOID: strings.Repeat("1", 40),
			Actions: []managedartifact.Action{{
				Type: managedartifact.ActionWriteREADME,
				Path: managedartifact.READMEPath,
				Observed: managedartifact.ObservedState{
					Present: &present,
					Mode:    "100644",
					SHA256:  strings.Repeat("a", 64),
				},
				Desired: managedartifact.DesiredState{
					Mode:    "100644",
					SHA256:  managedartifact.DigestSHA256([]byte(desired)),
					Content: &desired,
				},
				TemplateSHA256: strings.Repeat("c", 64),
				Diff:           "--- a/README.md\n+++ b/README.md\n@@ review @@\n-\"old\\n\"\n+\"new\\n\"\n",
			}},
		}},
	}
	plan.Repositories[0].Actions[0].Desired.SHA256 = strings.Repeat("b", 64)
	return plan
}

func writeManagedPlanConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "repora.yaml")
	data := []byte(`repos:
  - id: demo
    uid: repo.demo
    canonical:
      provider: gitlab
      path: example/demo
    mirrors:
      - provider: github
        path: example/demo
    artifacts:
      readme:
        template: templates/README.md.tmpl
`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func captureManagedStdout(t *testing.T, dst io.Writer, fn func() int) int {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	code := fn()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(dst, r); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return code
}

func captureManagedStderr(t *testing.T, dst io.Writer, fn func() int) int {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() { os.Stderr = old }()

	code := fn()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(dst, r); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return code
}

func TestManagedPlanFixtureJSONIsStrict(t *testing.T) {
	plan := managedPlanFixture(t)
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := managedartifact.ParsePlan(data); err == nil {
		t.Fatal("fixture should fail strict digest validation until test uses real desired digest")
	}
}
