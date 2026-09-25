package posture

import (
	"testing"
)

func TestCIEnvironmentV2HostToolAndSetupSignalsAreBounded(t *testing.T) {
	data := []byte(`name: ci
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      # run: go test ./...
      - uses: actions/setup-go@v6
      - name: harmless prose
        env:
          NOTE: "go test and actions/setup-python@v5"
        run: |
          echo "go test"
          go test ./...
          jq . result.json
      - run: nix develop .#ci --command cargo test
`)
	runs, uses, err := extractCIWorkflowSignalInputs(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := detectHostToolInvocationSignals(runs); !equalStrings(got, []string{"go", "jq"}) {
		t.Fatalf("host-tool signals = %#v", got)
	}
	if got := detectToolSetupActionSignals(uses); !equalStrings(got, []string{"actions/setup-go"}) {
		t.Fatalf("setup-action signals = %#v", got)
	}
}

func TestCIEnvironmentV2MalformedWorkflowLeavesNewSignalsUnknown(t *testing.T) {
	if _, _, err := extractCIWorkflowSignalInputs([]byte("jobs: [")); err == nil {
		t.Fatal("malformed workflow unexpectedly parsed")
	}
}

func TestCIEnvironmentInventoryVersionsEnforceSignalShape(t *testing.T) {
	base := CIEnvironmentInventory{
		Kind:                  CIEnvironmentInventoryKind,
		Repository:            RepositoryIdentity{Provider: "github", FullName: "acme/project"},
		DefaultBranch:         Observed("main"),
		DefaultCommit:         Observed("abc"),
		ProfileDeclared:       Observed(false),
		DeclaredApplicability: Unknown[string](),
		FlakePresent:          Observed(true),
		FlakeLockPresent:      Observed(true),
		WorkflowsState:        StateObserved,
		ExternalInputs:        []CIExternalInputFact{},
		Evidence:              []Evidence{},
	}
	workflow := CIEnvironmentWorkflowFact{
		Path: ".github/workflows/ci.yml",
		ContentState: StateObserved,
		FlakeInvocationSignals: Observed([]string{}),
		ImperativeInstallSignals: Observed([]string{}),
		Evidence: []Evidence{},
	}

	legacy := base
	legacy.Version = CIEnvironmentInventoryVersion
	legacy.Workflows = []CIEnvironmentWorkflowFact{workflow}
	if err := legacy.Validate(); err != nil {
		t.Fatalf("v1 rejected unchanged workflow shape: %v", err)
	}

	host := Observed([]string{"go"})
	setup := Observed([]string{"actions/setup-go"})
	current := base
	current.Version = CIEnvironmentInventoryVersionV2
	workflow.HostToolInvocationSignals = &host
	workflow.ToolSetupActionSignals = &setup
	current.Workflows = []CIEnvironmentWorkflowFact{workflow}
	if err := current.Validate(); err != nil {
		t.Fatalf("v2 rejected required tool signals: %v", err)
	}

	current.Workflows[0].HostToolInvocationSignals = nil
	if err := current.Validate(); err == nil {
		t.Fatal("v2 accepted missing host-tool signal fact")
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}
