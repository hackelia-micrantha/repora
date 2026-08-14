package posture

import "testing"

func TestParseWorkflowNormalizesPermissionsActionsAndRunners(t *testing.T) {
	data := []byte(`name: ci
on:
  pull_request:
  pull_request_target:
permissions: read-all
jobs:
  build:
    runs-on: [self-hosted, linux, x64]
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@11d5960a326750d5838078e36cf38b85af677262
      - uses: acme/tool@v1
  dynamic:
    runs-on: ${{ matrix.runner }}
    uses: hackelia-micrantha/repora/.github/workflows/reusable.yml@main
`)
	evidence := Evidence{Source: "github.workflow", Reference: ".github/workflows/ci.yml"}
	workflow, err := parseWorkflow("hackelia-micrantha/repora", ".github/workflows/ci.yml", data, evidence)
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	if !workflow.UsesPullRequestTarget {
		t.Fatal("pull_request_target was not detected")
	}
	if !workflow.Permissions.Declared || workflow.Permissions.Default != "read-all" {
		t.Fatalf("workflow permissions = %#v", workflow.Permissions)
	}
	if len(workflow.Jobs) != 2 || workflow.Jobs[0].Name != "build" || workflow.Jobs[1].Name != "dynamic" {
		t.Fatalf("jobs = %#v", workflow.Jobs)
	}
	build := workflow.Jobs[0]
	if build.SelfHosted.State != StateObserved || build.SelfHosted.Value == nil || !*build.SelfHosted.Value {
		t.Fatalf("build self-hosted fact = %#v", build.SelfHosted)
	}
	if len(build.Permissions.Scopes) != 1 || build.Permissions.Scopes[0].Scope != "contents" || build.Permissions.Scopes[0].Access != "read" {
		t.Fatalf("build permissions = %#v", build.Permissions)
	}
	if len(build.Actions) != 2 {
		t.Fatalf("build actions = %#v", build.Actions)
	}
	if build.Actions[0].ThirdParty || build.Actions[0].Pinning != "immutable-sha" {
		t.Fatalf("checkout action = %#v", build.Actions[0])
	}
	if !build.Actions[1].ThirdParty || build.Actions[1].Pinning != "mutable-ref" {
		t.Fatalf("third-party action = %#v", build.Actions[1])
	}
	dynamic := workflow.Jobs[1]
	if dynamic.SelfHosted.State != StateUnknown || dynamic.SelfHosted.Value != nil {
		t.Fatalf("dynamic runner fact = %#v", dynamic.SelfHosted)
	}
	if len(dynamic.Actions) != 1 || dynamic.Actions[0].ThirdParty || dynamic.Actions[0].Pinning != "mutable-ref" {
		t.Fatalf("reusable workflow action = %#v", dynamic.Actions)
	}
}

func TestParseWorkflowTreatsOmittedPermissionsAsUndeclared(t *testing.T) {
	workflow, err := parseWorkflow("owner/repo", ".github/workflows/test.yml", []byte(`on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo ok
`), Evidence{Source: "github.workflow", Reference: ".github/workflows/test.yml"})
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	if workflow.Permissions.Declared {
		t.Fatal("omitted workflow permissions reported as declared")
	}
	if len(workflow.Jobs) != 1 {
		t.Fatalf("jobs = %d", len(workflow.Jobs))
	}
	fact := workflow.Jobs[0].SelfHosted
	if fact.State != StateObserved || fact.Value == nil || *fact.Value {
		t.Fatalf("hosted runner fact = %#v", fact)
	}
}
