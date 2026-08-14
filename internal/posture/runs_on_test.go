package posture

import "testing"

func TestParseWorkflowNormalizesRunsOnMappingLabels(t *testing.T) {
	workflow, err := parseWorkflow("owner/repo", ".github/workflows/group.yml", []byte(`on: push
jobs:
  grouped:
    runs-on:
      group: build-runners
      labels: [self-hosted, linux, x64]
    steps:
      - run: echo ok
`), Evidence{Source: "github.workflow", Reference: ".github/workflows/group.yml"})
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	if len(workflow.Jobs) != 1 {
		t.Fatalf("jobs = %d", len(workflow.Jobs))
	}
	job := workflow.Jobs[0]
	if len(job.RunsOn) != 3 || job.RunsOn[0] != "self-hosted" || job.RunsOn[1] != "linux" || job.RunsOn[2] != "x64" {
		t.Fatalf("runs-on labels = %#v", job.RunsOn)
	}
	if job.SelfHosted.State != StateObserved || job.SelfHosted.Value == nil || !*job.SelfHosted.Value {
		t.Fatalf("self-hosted fact = %#v", job.SelfHosted)
	}
}

func TestParseWorkflowTreatsGroupOnlyRunnerAsUnknownSelfHosted(t *testing.T) {
	workflow, err := parseWorkflow("owner/repo", ".github/workflows/group.yml", []byte(`on: push
jobs:
  grouped:
    runs-on:
      group: build-runners
    steps:
      - run: echo ok
`), Evidence{Source: "github.workflow", Reference: ".github/workflows/group.yml"})
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	fact := workflow.Jobs[0].SelfHosted
	if fact.State != StateUnknown || fact.Value != nil {
		t.Fatalf("group-only runner fact = %#v", fact)
	}
}
