package posture

import (
	"context"
	"strings"
	"testing"
)

func TestCollectGitHubCIEnvironmentCapturesFlakeAndInstallSignals(t *testing.T) {
	reader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repo"),
		branch:        GitHubBranch{Name: "main", CommitSHA: "abc", TreeSHA: "tree"},
		branchObs:     available("github.branch", "main"),
		tree: GitHubTree{Entries: []GitHubTreeEntry{
			{Path: ciEnvironmentProfilePath, Type: "blob", SHA: "profile"},
			{Path: "flake.nix", Type: "blob", SHA: "flake"},
			{Path: "flake.lock", Type: "blob", SHA: "lock"},
			{Path: ".github/workflows/ci.yml", Type: "blob", SHA: "ci"},
		}},
		treeObs: available("github.git_tree", "tree"),
		blobs: map[string][]byte{
			"profile": []byte("kind: repora.posture-ci-environment-profile\nversion: 1\nci_applicability: applicable\nexternal_inputs:\n  - id: linux-kernel\n    class: platform\n    rationale: required by the runner isolation boundary\n"),
			"ci":      []byte("name: CI\njobs:\n  test:\n    steps:\n      - uses: actions/setup-go@v6\n      - run: go test ./...\n      - run: nix flake check\n      - run: apt-get install -y shellcheck\n"),
		},
	}
	inventory, err := CollectGitHubCIEnvironment(context.Background(), reader, "acme/project")
	if err != nil {
		t.Fatal(err)
	}
	if !boolValue(t, inventory.FlakePresent) || !boolValue(t, inventory.FlakeLockPresent) {
		t.Fatalf("flake facts = %#v %#v", inventory.FlakePresent, inventory.FlakeLockPresent)
	}
	if inventory.DeclaredApplicability.Value == nil || *inventory.DeclaredApplicability.Value != "applicable" {
		t.Fatalf("applicability = %#v", inventory.DeclaredApplicability)
	}
	if len(inventory.ExternalInputs) != 1 || inventory.ExternalInputs[0].ID != "linux-kernel" {
		t.Fatalf("external inputs = %#v", inventory.ExternalInputs)
	}
	if len(inventory.Workflows) != 1 {
		t.Fatalf("workflows = %#v", inventory.Workflows)
	}
	workflow := inventory.Workflows[0]
	if workflow.FlakeInvocationSignals.Value == nil || len(*workflow.FlakeInvocationSignals.Value) != 1 || (*workflow.FlakeInvocationSignals.Value)[0] != "flake-check" {
		t.Fatalf("flake signals = %#v", workflow.FlakeInvocationSignals)
	}
	if workflow.ImperativeInstallSignals.Value == nil || len(*workflow.ImperativeInstallSignals.Value) != 1 || (*workflow.ImperativeInstallSignals.Value)[0] != "apt-install" {
		t.Fatalf("install signals = %#v", workflow.ImperativeInstallSignals)
	}
	if workflow.WorkloadToolSignals.Value == nil || len(*workflow.WorkloadToolSignals.Value) != 1 || (*workflow.WorkloadToolSignals.Value)[0] != "go" {
		t.Fatalf("workload tool signals = %#v", workflow.WorkloadToolSignals)
	}
	if workflow.SetupProvisioningSignals.Value == nil || len(*workflow.SetupProvisioningSignals.Value) != 1 || (*workflow.SetupProvisioningSignals.Value)[0] != "go" {
		t.Fatalf("setup signals = %#v", workflow.SetupProvisioningSignals)
	}
	if workflow.AmbientToolCandidates.Value == nil || len(*workflow.AmbientToolCandidates.Value) != 0 {
		t.Fatalf("ambient candidates = %#v", workflow.AmbientToolCandidates)
	}
}

func TestDetectWorkflowToolSignalsFindsBareAmbientCandidates(t *testing.T) {
	data := []byte(`name: CI
jobs:
  test:
    steps:
      - run: go test ./...
      - run: |
          # python in a comment must not count
          python3 -m pytest
      - run: nix develop .#ci -c cargo test
`)
	tools, setups := detectWorkflowToolSignals(data)
	if got, want := strings.Join(tools, ","), "go,python"; got != want {
		t.Fatalf("tools = %q, want %q", got, want)
	}
	if len(setups) != 0 {
		t.Fatalf("setups = %#v, want none", setups)
	}
	if got, want := strings.Join(detectAmbientToolCandidates(tools, setups), ","), "go,python"; got != want {
		t.Fatalf("ambient candidates = %q, want %q", got, want)
	}
}

func TestDetectWorkflowToolSignalsCorrelatesKnownSetupActions(t *testing.T) {
	data := []byte(`name: CI
jobs:
  test:
    steps:
      - uses: actions/setup-python@v6
      - uses: actions/setup-node@v5
      - run: python -m pytest
      - run: npm test
`)
	tools, setups := detectWorkflowToolSignals(data)
	if got, want := strings.Join(tools, ","), "node,python"; got != want {
		t.Fatalf("tools = %q, want %q", got, want)
	}
	if got, want := strings.Join(setups, ","), "node,python"; got != want {
		t.Fatalf("setups = %q, want %q", got, want)
	}
	if candidates := detectAmbientToolCandidates(tools, setups); len(candidates) != 0 {
		t.Fatalf("ambient candidates = %#v, want none", candidates)
	}
}

func TestCollectGitHubCIEnvironmentDoesNotInferApplicability(t *testing.T) {
	reader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repo"),
		branch:        GitHubBranch{Name: "main", CommitSHA: "abc", TreeSHA: "tree"},
		branchObs:     available("github.branch", "main"),
		tree: GitHubTree{Entries: []GitHubTreeEntry{
			{Path: ".github/workflows/ci.yml", Type: "blob", SHA: "ci"},
		}},
		treeObs: available("github.git_tree", "tree"),
		blobs:   map[string][]byte{"ci": []byte("name: CI\n")},
	}
	inventory, err := CollectGitHubCIEnvironment(context.Background(), reader, "acme/project")
	if err != nil {
		t.Fatal(err)
	}
	if inventory.DeclaredApplicability.State != StateUnknown {
		t.Fatalf("workflow presence inferred applicability: %#v", inventory.DeclaredApplicability)
	}
	if inventory.FlakePresent.State != StateObserved || boolValue(t, inventory.FlakePresent) {
		t.Fatalf("missing flake fact = %#v", inventory.FlakePresent)
	}
}

func TestParseCIEnvironmentProfileRejectsDuplicateExternalInput(t *testing.T) {
	_, err := ParseCIEnvironmentProfile([]byte("kind: repora.posture-ci-environment-profile\nversion: 1\nci_applicability: applicable\nexternal_inputs:\n  - id: xcode\n    class: platform\n    rationale: vendor platform\n  - id: xcode\n    class: bootstrap\n    rationale: duplicate\n"))
	if err == nil {
		t.Fatal("duplicate external input accepted")
	}
}

func TestCollectGitHubCIEnvironmentMalformedProfilePreservesUnknownDeclaration(t *testing.T) {
	reader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repo"),
		branch:        GitHubBranch{Name: "main", CommitSHA: "abc", TreeSHA: "tree"},
		branchObs:     available("github.branch", "main"),
		tree: GitHubTree{Entries: []GitHubTreeEntry{
			{Path: ciEnvironmentProfilePath, Type: "blob", SHA: "profile"},
		}},
		treeObs: available("github.git_tree", "tree"),
		blobs:   map[string][]byte{"profile": []byte("kind: wrong\nversion: 1\nci_applicability: applicable\n")},
	}
	inventory, err := CollectGitHubCIEnvironment(context.Background(), reader, "acme/project")
	if err != nil {
		t.Fatal(err)
	}
	if !boolValue(t, inventory.ProfileDeclared) || inventory.DeclaredApplicability.State != StateUnknown {
		t.Fatalf("malformed profile facts = %#v %#v", inventory.ProfileDeclared, inventory.DeclaredApplicability)
	}
}
