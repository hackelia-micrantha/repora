package posturepolicy

import (
	"testing"

	"repoctl/internal/posture"
)

func TestAddCIEnvironmentPreservesDeclarationsAndSignals(t *testing.T) {
	inventory := posture.CIEnvironmentInventory{
		Kind:                  posture.CIEnvironmentInventoryKind,
		Version:               posture.CIEnvironmentInventoryVersion,
		Repository:            posture.RepositoryIdentity{Provider: "github", FullName: "acme/project"},
		DefaultBranch:         posture.Observed("main"),
		DefaultCommit:         posture.Observed("abc1234"),
		ProfileDeclared:       posture.Observed(true),
		DeclaredApplicability: posture.Observed("applicable"),
		FlakePresent:          posture.Observed(true),
		FlakeLockPresent:      posture.Observed(true),
		WorkflowsState:        posture.StateObserved,
		Workflows: []posture.CIEnvironmentWorkflowFact{
			{
				Path:                     ".github/workflows/ci.yml",
				ContentState:             posture.StateObserved,
				FlakeInvocationSignals:   posture.Observed([]string{"flake-check"}),
				ImperativeInstallSignals: posture.Observed([]string{"apt-install"}),
				Evidence:                 []posture.Evidence{},
			},
		},
		ExternalInputs: []posture.CIExternalInputFact{
			{ID: "linux-kernel", Class: "platform", Rationale: "runner boundary", Evidence: []posture.Evidence{}},
		},
		Evidence: []posture.Evidence{},
	}
	inputs := NewInputs("acme/project")
	if err := AddCIEnvironment(&inputs, inventory); err != nil {
		t.Fatal(err)
	}
	if got := string(inputs.Facts["ci_environment.declared_applicability"].Value); got != `"applicable"` {
		t.Fatalf("applicability = %s", got)
	}
	if got := string(inputs.Facts["ci_environment.workflow_count"].Value); got != "1" {
		t.Fatalf("workflow count = %s", got)
	}
	if got := string(inputs.Facts["ci_environment.workflows_with_flake_signals_count"].Value); got != "1" {
		t.Fatalf("flake workflow count = %s", got)
	}
	if got := string(inputs.Facts["ci_environment.external_input.linux-kernel.class"].Value); got != `"platform"` {
		t.Fatalf("external input class = %s", got)
	}
}

func TestConvergeCIEnvironmentArtifact(t *testing.T) {
	inventory := posture.CIEnvironmentInventory{
		Kind:                  posture.CIEnvironmentInventoryKind,
		Version:               posture.CIEnvironmentInventoryVersion,
		Repository:            posture.RepositoryIdentity{Provider: "github", FullName: "acme/project"},
		DefaultBranch:         posture.Observed("main"),
		DefaultCommit:         posture.Observed("abc1234"),
		ProfileDeclared:       posture.Observed(false),
		DeclaredApplicability: posture.Unknown[string](),
		FlakePresent:          posture.Observed(false),
		FlakeLockPresent:      posture.Observed(false),
		WorkflowsState:        posture.StateObserved,
		Workflows:             []posture.CIEnvironmentWorkflowFact{},
		ExternalInputs:        []posture.CIExternalInputFact{},
		Evidence:              []posture.Evidence{},
	}
	data, err := inventory.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	inputs, err := ConvergeArtifacts(ArtifactSet{CIEnvironment: data})
	if err != nil {
		t.Fatal(err)
	}
	if inputs.Facts["ci_environment.declared_applicability"].State != posture.StateUnknown {
		t.Fatalf("applicability state = %q", inputs.Facts["ci_environment.declared_applicability"].State)
	}
}
