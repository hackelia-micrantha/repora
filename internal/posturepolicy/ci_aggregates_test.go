package posturepolicy

import (
	"encoding/json"
	"testing"

	"repoctl/internal/posture"
)

var ciAggregateFacts = []string{
	"ci.workflow_count",
	"ci.third_party_action_count",
	"ci.mutable_third_party_action_count",
	"ci.pull_request_target_workflow_count",
	"ci.workflows_without_declared_permissions_count",
}

func TestAddInventoryAddsStableCIAggregates(t *testing.T) {
	inventory := posture.NewInventory("acme/project")
	inventory.RepositoryFacts = validRepositoryFactsForAdapterTest()
	inventory.WorkflowsState = posture.StateObserved
	inventory.Workflows = []posture.Workflow{
		observedWorkflow(
			".github/workflows/ci.yml",
			true,
			false,
			[]posture.ActionReference{
				{Uses: "actions/checkout@0123456789012345678901234567890123456789", ThirdParty: true, Pinning: "immutable-sha"},
				{Uses: "./.github/actions/local", ThirdParty: false, Pinning: "local"},
			},
		),
		observedWorkflow(
			".github/workflows/release.yml",
			false,
			true,
			[]posture.ActionReference{
				{Uses: "vendor/action@v1", ThirdParty: true, Pinning: "mutable-ref"},
				{Uses: "docker://example/image@sha256:deadbeef", ThirdParty: true, Pinning: "immutable-digest"},
			},
		),
	}

	inputs := NewInputs("acme/project")
	if err := AddInventory(&inputs, inventory); err != nil {
		t.Fatal(err)
	}

	assertObservedInt(t, inputs, "ci.workflow_count", 2)
	assertObservedInt(t, inputs, "ci.third_party_action_count", 3)
	assertObservedInt(t, inputs, "ci.mutable_third_party_action_count", 1)
	assertObservedInt(t, inputs, "ci.pull_request_target_workflow_count", 1)
	assertObservedInt(t, inputs, "ci.workflows_without_declared_permissions_count", 1)
}

func TestAddInventoryCIAggregatesAreZeroForObservedEmptyWorkflowSet(t *testing.T) {
	inventory := posture.NewInventory("acme/project")
	inventory.RepositoryFacts = validRepositoryFactsForAdapterTest()
	inventory.WorkflowsState = posture.StateObserved
	inventory.Workflows = []posture.Workflow{}

	inputs := NewInputs("acme/project")
	if err := AddInventory(&inputs, inventory); err != nil {
		t.Fatal(err)
	}
	for _, name := range ciAggregateFacts {
		assertObservedInt(t, inputs, name, 0)
	}
}

func TestAddInventoryCIAggregatesPreserveIncompleteWorkflowEvidence(t *testing.T) {
	cases := []struct {
		name           string
		workflowsState posture.FactState
		workflows      []posture.Workflow
		wantState      posture.FactState
	}{
		{name: "unknown inventory", workflowsState: posture.StateUnknown, workflows: []posture.Workflow{}, wantState: posture.StateUnknown},
		{name: "unavailable inventory", workflowsState: posture.StateUnavailable, workflows: []posture.Workflow{}, wantState: posture.StateUnavailable},
		{name: "unknown workflow", workflowsState: posture.StateObserved, workflows: []posture.Workflow{incompleteWorkflow(posture.StateUnknown)}, wantState: posture.StateUnknown},
		{name: "unavailable workflow", workflowsState: posture.StateObserved, workflows: []posture.Workflow{incompleteWorkflow(posture.StateUnavailable)}, wantState: posture.StateUnavailable},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inventory := posture.NewInventory("acme/project")
			inventory.RepositoryFacts = validRepositoryFactsForAdapterTest()
			inventory.WorkflowsState = tc.workflowsState
			inventory.Workflows = tc.workflows

			inputs := NewInputs("acme/project")
			if err := AddInventory(&inputs, inventory); err != nil {
				t.Fatal(err)
			}
			for _, name := range ciAggregateFacts {
				fact, ok := inputs.Facts[name]
				if !ok {
					t.Fatalf("missing aggregate fact %q", name)
				}
				if fact.State != tc.wantState {
					t.Fatalf("%s state = %q, want %q", name, fact.State, tc.wantState)
				}
				if len(fact.Value) != 0 {
					t.Fatalf("%s incomplete state carried value %s", name, fact.Value)
				}
			}
		})
	}
}

func observedWorkflow(path string, permissionsDeclared, pullRequestTarget bool, actions []posture.ActionReference) posture.Workflow {
	return posture.Workflow{
		Path:                  path,
		State:                 posture.StateObserved,
		Permissions:           posture.Permissions{Declared: permissionsDeclared, Scopes: []posture.PermissionScope{}},
		UsesPullRequestTarget: pullRequestTarget,
		Jobs: []posture.WorkflowJob{{
			Name:            "build",
			Permissions:     posture.Permissions{Scopes: []posture.PermissionScope{}},
			RunsOn:          []string{"ubuntu-latest"},
			SelfHostedLabel: posture.Observed(false),
			Actions:         actions,
		}},
		Evidence: []posture.Evidence{},
	}
}

func incompleteWorkflow(state posture.FactState) posture.Workflow {
	return posture.Workflow{
		Path:        ".github/workflows/incomplete.yml",
		State:       state,
		Permissions: posture.Permissions{Scopes: []posture.PermissionScope{}},
		Jobs:        []posture.WorkflowJob{},
		Evidence:    []posture.Evidence{{Source: "github.workflow", Reference: ".github/workflows/incomplete.yml"}},
	}
}

func assertObservedInt(t *testing.T, inputs Inputs, name string, want int) {
	t.Helper()
	fact, ok := inputs.Facts[name]
	if !ok {
		t.Fatalf("missing aggregate fact %q", name)
	}
	if fact.State != posture.StateObserved {
		t.Fatalf("%s state = %q, want observed", name, fact.State)
	}
	var got int
	if err := json.Unmarshal(fact.Value, &got); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	if got != want {
		t.Fatalf("%s = %d, want %d", name, got, want)
	}
}
