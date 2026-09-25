package posture

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	CIEnvironmentInventoryKind      = "repora.posture-ci-environment"
	CIEnvironmentInventoryVersionV1 = 1
	CIEnvironmentInventoryVersion   = 2
	CIEnvironmentProfileKind        = "repora.posture-ci-environment-profile"
	CIEnvironmentProfileVersion     = 1
	ciEnvironmentProfilePath        = ".repora/posture-ci-environment.yaml"
	maxCIEnvironmentBytes           = 1 << 20
	maxCIExternalInputs             = 64
)

type CIEnvironmentProfile struct {
	Kind            string                       `json:"kind" yaml:"kind"`
	Version         int                          `json:"version" yaml:"version"`
	CIApplicability string                       `json:"ci_applicability" yaml:"ci_applicability"`
	ExternalInputs  []CIExternalInputDeclaration `json:"external_inputs" yaml:"external_inputs"`
}

type CIExternalInputDeclaration struct {
	ID        string `json:"id" yaml:"id"`
	Class     string `json:"class" yaml:"class"`
	Rationale string `json:"rationale" yaml:"rationale"`
}

type CIExternalInputFact struct {
	ID        string     `json:"id"`
	Class     string     `json:"class"`
	Rationale string     `json:"rationale"`
	Evidence  []Evidence `json:"evidence"`
}

type CIEnvironmentWorkflowFact struct {
	Path                     string         `json:"path"`
	ContentState             FactState      `json:"content_state"`
	FlakeInvocationSignals   Fact[[]string] `json:"flake_invocation_signals"`
	ImperativeInstallSignals Fact[[]string] `json:"imperative_install_signals"`
	WorkloadToolSignals      Fact[[]string] `json:"workload_tool_signals,omitempty"`
	SetupProvisioningSignals Fact[[]string] `json:"setup_provisioning_signals,omitempty"`
	AmbientToolCandidates    Fact[[]string] `json:"ambient_tool_candidates,omitempty"`
	Evidence                 []Evidence     `json:"evidence"`
}

type CIEnvironmentInventory struct {
	Kind                  string                      `json:"kind"`
	Version               int                         `json:"version"`
	Repository            RepositoryIdentity          `json:"repository"`
	DefaultBranch         Fact[string]                `json:"default_branch"`
	DefaultCommit         Fact[string]                `json:"default_commit"`
	ProfileDeclared       Fact[bool]                  `json:"profile_declared"`
	DeclaredApplicability Fact[string]                `json:"declared_applicability"`
	FlakePresent          Fact[bool]                  `json:"flake_present"`
	FlakeLockPresent      Fact[bool]                  `json:"flake_lock_present"`
	WorkflowsState        FactState                   `json:"workflows_state"`
	Workflows             []CIEnvironmentWorkflowFact `json:"workflows"`
	ExternalInputs        []CIExternalInputFact       `json:"external_inputs"`
	Evidence              []Evidence                  `json:"evidence"`
}

func ParseCIEnvironmentProfile(data []byte) (CIEnvironmentProfile, error) {
	if len(data) > maxCIEnvironmentBytes {
		return CIEnvironmentProfile{}, fmt.Errorf("CI environment profile exceeds %d bytes", maxCIEnvironmentBytes)
	}
	var profile CIEnvironmentProfile
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&profile); err != nil {
		return CIEnvironmentProfile{}, fmt.Errorf("parse CI environment profile: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return CIEnvironmentProfile{}, fmt.Errorf("CI environment profile must contain exactly one YAML document")
		}
		return CIEnvironmentProfile{}, fmt.Errorf("parse CI environment profile trailing content: %w", err)
	}
	if err := profile.Validate(); err != nil {
		return CIEnvironmentProfile{}, err
	}
	sort.Slice(profile.ExternalInputs, func(i, j int) bool { return profile.ExternalInputs[i].ID < profile.ExternalInputs[j].ID })
	return profile, nil
}

func (p CIEnvironmentProfile) Validate() error {
	if p.Kind != CIEnvironmentProfileKind || p.Version != CIEnvironmentProfileVersion {
		return fmt.Errorf("unsupported CI environment profile contract: kind=%q version=%d", p.Kind, p.Version)
	}
	switch p.CIApplicability {
	case "applicable", "not-applicable", "unresolved":
	default:
		return fmt.Errorf("ci_applicability must be applicable, not-applicable, or unresolved")
	}
	if len(p.ExternalInputs) > maxCIExternalInputs {
		return fmt.Errorf("CI environment profile exceeds %d external inputs", maxCIExternalInputs)
	}
	seen := map[string]struct{}{}
	for idx, input := range p.ExternalInputs {
		if !validCIExternalInputID(input.ID) {
			return fmt.Errorf("external_inputs[%d] id is invalid", idx)
		}
		if _, exists := seen[input.ID]; exists {
			return fmt.Errorf("external input %q is duplicated", input.ID)
		}
		seen[input.ID] = struct{}{}
		if input.Class != "bootstrap" && input.Class != "platform" {
			return fmt.Errorf("external input %q class must be bootstrap or platform", input.ID)
		}
		if strings.TrimSpace(input.Rationale) == "" {
			return fmt.Errorf("external input %q rationale is required", input.ID)
		}
	}
	return nil
}

func validCIExternalInputID(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func newCIEnvironmentInventory(fullName string) CIEnvironmentInventory {
	return CIEnvironmentInventory{
		Kind:           CIEnvironmentInventoryKind,
		Version:        CIEnvironmentInventoryVersion,
		Repository:     RepositoryIdentity{Provider: "github", FullName: fullName},
		Workflows:      []CIEnvironmentWorkflowFact{},
		ExternalInputs: []CIExternalInputFact{},
		Evidence:       []Evidence{},
	}
}

func (i CIEnvironmentInventory) Validate() error {
	if i.Kind != CIEnvironmentInventoryKind || (i.Version != CIEnvironmentInventoryVersionV1 && i.Version != CIEnvironmentInventoryVersion) {
		return fmt.Errorf("unsupported CI environment inventory contract: kind=%q version=%d", i.Kind, i.Version)
	}
	if i.Repository.Provider != "github" {
		return fmt.Errorf("CI environment inventory provider must be github")
	}
	if _, _, err := splitGitHubFullName(i.Repository.FullName); err != nil {
		return err
	}
	checks := []error{
		validateFact("default_branch", i.DefaultBranch),
		validateFact("default_commit", i.DefaultCommit),
		validateFact("profile_declared", i.ProfileDeclared),
		validateFact("declared_applicability", i.DeclaredApplicability),
		validateFact("flake_present", i.FlakePresent),
		validateFact("flake_lock_present", i.FlakeLockPresent),
	}
	for _, err := range checks {
		if err != nil {
			return err
		}
	}
	if i.DeclaredApplicability.State == StateObserved && i.DeclaredApplicability.Value != nil {
		switch *i.DeclaredApplicability.Value {
		case "applicable", "not-applicable", "unresolved":
		default:
			return fmt.Errorf("declared_applicability has unsupported value %q", *i.DeclaredApplicability.Value)
		}
	}
	if !validState(i.WorkflowsState) {
		return fmt.Errorf("workflows_state %q is invalid", i.WorkflowsState)
	}
	if i.Workflows == nil || i.ExternalInputs == nil || i.Evidence == nil {
		return fmt.Errorf("CI environment inventory arrays are required")
	}
	for idx, workflow := range i.Workflows {
		if err := validateDocumentationPath(workflow.Path); err != nil {
			return fmt.Errorf("workflows[%d]: %w", idx, err)
		}
		if !validState(workflow.ContentState) {
			return fmt.Errorf("workflow %q content_state %q is invalid", workflow.Path, workflow.ContentState)
		}
		if err := validateFact("flake_invocation_signals", workflow.FlakeInvocationSignals); err != nil {
			return err
		}
		if err := validateFact("imperative_install_signals", workflow.ImperativeInstallSignals); err != nil {
			return err
		}
		if i.Version >= CIEnvironmentInventoryVersion {
			if err := validateFact("workload_tool_signals", workflow.WorkloadToolSignals); err != nil {
				return err
			}
			if err := validateFact("setup_provisioning_signals", workflow.SetupProvisioningSignals); err != nil {
				return err
			}
			if err := validateFact("ambient_tool_candidates", workflow.AmbientToolCandidates); err != nil {
				return err
			}
		}
		if workflow.Evidence == nil {
			return fmt.Errorf("workflow %q evidence array is required", workflow.Path)
		}
	}
	seenInputs := map[string]struct{}{}
	for idx, input := range i.ExternalInputs {
		if !validCIExternalInputID(input.ID) {
			return fmt.Errorf("external_inputs[%d] id is invalid", idx)
		}
		if _, exists := seenInputs[input.ID]; exists {
			return fmt.Errorf("external input %q is duplicated", input.ID)
		}
		seenInputs[input.ID] = struct{}{}
		if input.Class != "bootstrap" && input.Class != "platform" {
			return fmt.Errorf("external input %q class is invalid", input.ID)
		}
		if strings.TrimSpace(input.Rationale) == "" || input.Evidence == nil {
			return fmt.Errorf("external input %q rationale and evidence are required", input.ID)
		}
	}
	return nil
}

func (i CIEnvironmentInventory) Marshal() ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode CI environment posture inventory: %w", err)
	}
	return append(data, '\n'), nil
}

func CollectGitHubCIEnvironment(ctx context.Context, reader GitHubReader, fullName string) (CIEnvironmentInventory, error) {
	if _, _, err := splitGitHubFullName(fullName); err != nil {
		return CIEnvironmentInventory{}, err
	}
	inventory := newCIEnvironmentInventory(fullName)
	repository, repositoryObs, err := reader.Repository(ctx, fullName)
	if err != nil {
		return CIEnvironmentInventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, repositoryObs.Evidence)
	if !repositoryObs.Available {
		setCIEnvironmentUnavailable(&inventory, repositoryObs.Evidence)
		return inventory, inventory.Validate()
	}
	inventory.DefaultBranch = Observed(repository.DefaultBranch, repositoryObs.Evidence)

	branch, branchObs, err := reader.Branch(ctx, fullName, repository.DefaultBranch)
	if err != nil {
		return CIEnvironmentInventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, branchObs.Evidence)
	if !branchObs.Available {
		setCIEnvironmentAfterBranchUnavailable(&inventory, branchObs.Evidence)
		return inventory, inventory.Validate()
	}
	inventory.DefaultCommit = Observed(branch.CommitSHA, branchObs.Evidence)

	tree, treeObs, err := reader.Tree(ctx, fullName, branch.TreeSHA)
	if err != nil {
		return CIEnvironmentInventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, treeObs.Evidence)
	if !treeObs.Available {
		setCIEnvironmentAfterTreeUnavailable(&inventory, treeObs.Evidence)
		return inventory, inventory.Validate()
	}

	entries := map[string]GitHubTreeEntry{}
	for _, entry := range tree.Entries {
		entries[entry.Path] = entry
	}
	inventory.ProfileDeclared = presenceFact(entries, tree, ciEnvironmentProfilePath, treeObs.Evidence)
	inventory.FlakePresent = presenceFact(entries, tree, "flake.nix", treeObs.Evidence)
	inventory.FlakeLockPresent = presenceFact(entries, tree, "flake.lock", treeObs.Evidence)

	profile, profileState, profileEvidence, err := loadCIEnvironmentProfile(ctx, reader, fullName, tree, entries, treeObs.Evidence)
	if err != nil {
		return CIEnvironmentInventory{}, err
	}
	if inventory.ProfileDeclared.State == StateObserved && inventory.ProfileDeclared.Value != nil && *inventory.ProfileDeclared.Value && profileState == StateObserved {
		inventory.DeclaredApplicability = Observed(profile.CIApplicability, profileEvidence)
		for _, input := range profile.ExternalInputs {
			inventory.ExternalInputs = append(inventory.ExternalInputs, CIExternalInputFact{
				ID: input.ID, Class: input.Class, Rationale: input.Rationale, Evidence: []Evidence{profileEvidence},
			})
		}
	} else {
		switch profileState {
		case StateUnavailable:
			inventory.DeclaredApplicability = Unavailable[string](profileEvidence)
		default:
			inventory.DeclaredApplicability = Unknown[string](evidenceWithDetail(profileEvidence, "no usable project CI applicability declaration observed"))
		}
	}

	inventory.Workflows, inventory.WorkflowsState, err = collectCIEnvironmentWorkflows(ctx, reader, fullName, tree, entries, treeObs.Evidence)
	if err != nil {
		return CIEnvironmentInventory{}, err
	}
	return inventory, inventory.Validate()
}

func loadCIEnvironmentProfile(ctx context.Context, reader GitHubReader, fullName string, tree GitHubTree, entries map[string]GitHubTreeEntry, treeEvidence Evidence) (CIEnvironmentProfile, FactState, Evidence, error) {
	entry, ok := entries[ciEnvironmentProfilePath]
	if !ok {
		if tree.Truncated {
			evidence := evidenceWithDetail(treeEvidence, "Git tree is truncated; CI environment profile presence is unknown")
			return CIEnvironmentProfile{}, StateUnknown, evidence, nil
		}
		return CIEnvironmentProfile{}, StateObserved, Evidence{Source: "repora.builtin", Reference: "ci-environment-profile:absent"}, nil
	}
	if entry.Type != "blob" {
		return CIEnvironmentProfile{}, StateUnknown, evidenceWithDetail(treeEvidence, "CI environment profile exists but is not a blob"), nil
	}
	data, obs, err := reader.Blob(ctx, fullName, entry.SHA)
	if err != nil {
		return CIEnvironmentProfile{}, "", Evidence{}, err
	}
	if !obs.Available {
		return CIEnvironmentProfile{}, StateUnavailable, obs.Evidence, nil
	}
	profile, err := ParseCIEnvironmentProfile(data)
	if err != nil {
		return CIEnvironmentProfile{}, StateUnknown, evidenceWithDetail(obs.Evidence, "declared CI environment profile is malformed or unsupported"), nil
	}
	return profile, StateObserved, obs.Evidence, nil
}

func collectCIEnvironmentWorkflows(ctx context.Context, reader GitHubReader, fullName string, tree GitHubTree, entries map[string]GitHubTreeEntry, treeEvidence Evidence) ([]CIEnvironmentWorkflowFact, FactState, error) {
	paths := []string{}
	for path, entry := range entries {
		if entry.Type == "blob" && strings.HasPrefix(path, ".github/workflows/") && (strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml")) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	state := StateObserved
	if tree.Truncated {
		state = StateUnknown
	}
	workflows := make([]CIEnvironmentWorkflowFact, 0, len(paths))
	for _, path := range paths {
		entry := entries[path]
		data, obs, err := reader.Blob(ctx, fullName, entry.SHA)
		if err != nil {
			return nil, "", err
		}
		fact := CIEnvironmentWorkflowFact{Path: path, Evidence: []Evidence{obs.Evidence}}
		if !obs.Available {
			fact.ContentState = StateUnavailable
			fact.FlakeInvocationSignals = Unavailable[[]string](obs.Evidence)
			fact.ImperativeInstallSignals = Unavailable[[]string](obs.Evidence)
			fact.WorkloadToolSignals = Unavailable[[]string](obs.Evidence)
			fact.SetupProvisioningSignals = Unavailable[[]string](obs.Evidence)
			fact.AmbientToolCandidates = Unavailable[[]string](obs.Evidence)
			workflows = append(workflows, fact)
			continue
		}
		if len(data) > maxWorkflowBytes {
			evidence := evidenceWithDetail(obs.Evidence, "workflow exceeds bounded static-inspection limit")
			fact.ContentState = StateUnknown
			fact.FlakeInvocationSignals = Unknown[[]string](evidence)
			fact.ImperativeInstallSignals = Unknown[[]string](evidence)
			fact.WorkloadToolSignals = Unknown[[]string](evidence)
			fact.SetupProvisioningSignals = Unknown[[]string](evidence)
			fact.AmbientToolCandidates = Unknown[[]string](evidence)
			workflows = append(workflows, fact)
			continue
		}
		fact.ContentState = StateObserved
		fact.FlakeInvocationSignals = Observed(detectFlakeInvocationSignals(data), obs.Evidence)
		fact.ImperativeInstallSignals = Observed(detectImperativeInstallSignals(data), obs.Evidence)
		toolSignals, setupSignals := detectWorkflowToolSignals(data)
		fact.WorkloadToolSignals = Observed(toolSignals, obs.Evidence)
		fact.SetupProvisioningSignals = Observed(setupSignals, obs.Evidence)
		fact.AmbientToolCandidates = Observed(detectAmbientToolCandidates(toolSignals, setupSignals), obs.Evidence)
		workflows = append(workflows, fact)
	}
	if tree.Truncated && len(paths) == 0 {
		_ = treeEvidence
	}
	return workflows, state, nil
}

var currentFlakeCommandPatterns = []struct {
	name string
	re   *regexp.Regexp
}{
	{name: "flake-check", re: regexp.MustCompile(`(?mi)\bnix[ \t]+flake[ \t]+check([ \t]|$)`)},
	{name: "current-flake-build", re: regexp.MustCompile(`(?mi)\bnix[ \t]+build[ \t]+\.(#|[ \t]|$)`)},
	{name: "current-flake-develop", re: regexp.MustCompile(`(?mi)\bnix[ \t]+develop[ \t]+\.(#|[ \t]|$)`)},
	{name: "current-flake-run", re: regexp.MustCompile(`(?mi)\bnix[ \t]+run[ \t]+\.(#|[ \t]|$)`)},
}

func detectFlakeInvocationSignals(data []byte) []string {
	text := string(data)
	signals := []string{}
	for _, pattern := range currentFlakeCommandPatterns {
		if pattern.re.MatchString(text) {
			signals = append(signals, pattern.name)
		}
	}
	return sortedUnique(signals)
}

type ciWorkflowDocument struct {
	Jobs map[string]struct {
		Steps []struct {
			Uses string `yaml:"uses"`
			Run  string `yaml:"run"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

var workloadToolNames = map[string]string{
	"go": "go",
	"python": "python",
	"python3": "python",
	"node": "node",
	"npm": "node",
	"pnpm": "node",
	"yarn": "node",
	"cargo": "rust",
	"rustc": "rust",
	"java": "java",
	"javac": "java",
	"dotnet": "dotnet",
	"gradle": "gradle",
	"mvn": "maven",
	"cmake": "cmake",
	"make": "make",
}

var setupActionSignals = []struct {
	prefix string
	signal string
}{
	{prefix: "actions/setup-go@", signal: "go"},
	{prefix: "actions/setup-python@", signal: "python"},
	{prefix: "actions/setup-node@", signal: "node"},
	{prefix: "actions/setup-java@", signal: "java"},
	{prefix: "actions/setup-dotnet@", signal: "dotnet"},
	{prefix: "dtolnay/rust-toolchain@", signal: "rust"},
	{prefix: "actions-rust-lang/setup-rust-toolchain@", signal: "rust"},
	{prefix: "gradle/actions/setup-gradle@", signal: "gradle"},
}

func detectWorkflowToolSignals(data []byte) ([]string, []string) {
	var document ciWorkflowDocument
	if err := yaml.Unmarshal(data, &document); err != nil {
		return []string{}, []string{}
	}
	tools := []string{}
	setups := []string{}
	for _, job := range document.Jobs {
		for _, step := range job.Steps {
			uses := strings.ToLower(strings.TrimSpace(step.Uses))
			for _, candidate := range setupActionSignals {
				if strings.HasPrefix(uses, candidate.prefix) {
					setups = append(setups, candidate.signal)
				}
			}
			for _, line := range strings.Split(step.Run, "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				fields := strings.Fields(line)
				if len(fields) == 0 {
					continue
				}
				command := strings.Trim(fields[0], "'\"")
				if strings.Contains(command, "/") {
					parts := strings.Split(command, "/")
					command = parts[len(parts)-1]
				}
				if family, ok := workloadToolNames[command]; ok {
					tools = append(tools, family)
				}
			}
		}
	}
	return sortedUnique(tools), sortedUnique(setups)
}

func detectAmbientToolCandidates(toolSignals, setupSignals []string) []string {
	provisioned := map[string]struct{}{}
	for _, signal := range setupSignals {
		provisioned[signal] = struct{}{}
	}
	candidates := []string{}
	for _, tool := range toolSignals {
		if _, ok := provisioned[tool]; !ok {
			candidates = append(candidates, tool)
		}
	}
	return sortedUnique(candidates)
}

var imperativeInstallPatterns = []struct {
	name string
	re   *regexp.Regexp
}{
	{name: "apt-install", re: regexp.MustCompile(`(?mi)\bapt(-get)?[ \t]+[^\n#]*\binstall\b`)},
	{name: "apk-add", re: regexp.MustCompile(`(?mi)\bapk[ \t]+add\b`)},
	{name: "dnf-install", re: regexp.MustCompile(`(?mi)\bdnf[ \t]+[^\n#]*\binstall\b`)},
	{name: "yum-install", re: regexp.MustCompile(`(?mi)\byum[ \t]+[^\n#]*\binstall\b`)},
	{name: "brew-install", re: regexp.MustCompile(`(?mi)\bbrew[ \t]+install\b`)},
	{name: "pip-install", re: regexp.MustCompile(`(?mi)(\bpython[0-9.]*[ \t]+-m[ \t]+pip|\bpip[0-9.]*)[ \t]+install\b`)},
	{name: "npm-global-install", re: regexp.MustCompile(`(?mi)\bnpm[ \t]+(install|i)[ \t]+(-g|--global)\b`)},
	{name: "pnpm-global-install", re: regexp.MustCompile(`(?mi)\bpnpm[ \t]+(add|install)[ \t]+(-g|--global)\b`)},
	{name: "yarn-global-add", re: regexp.MustCompile(`(?mi)\byarn[ \t]+global[ \t]+add\b`)},
	{name: "cargo-install", re: regexp.MustCompile(`(?mi)\bcargo[ \t]+install\b`)},
	{name: "go-install", re: regexp.MustCompile(`(?mi)\bgo[ \t]+install\b`)},
	{name: "rustup-install", re: regexp.MustCompile(`(?mi)\brustup[ \t]+(toolchain[ \t]+install|component[ \t]+add)\b`)},
}

func detectImperativeInstallSignals(data []byte) []string {
	text := string(data)
	signals := []string{}
	for _, pattern := range imperativeInstallPatterns {
		if pattern.re.MatchString(text) {
			signals = append(signals, pattern.name)
		}
	}
	return sortedUnique(signals)
}

func setCIEnvironmentUnavailable(i *CIEnvironmentInventory, evidence Evidence) {
	i.DefaultBranch = Unavailable[string](evidence)
	setCIEnvironmentAfterBranchUnavailable(i, evidence)
}

func setCIEnvironmentAfterBranchUnavailable(i *CIEnvironmentInventory, evidence Evidence) {
	i.DefaultCommit = Unavailable[string](evidence)
	setCIEnvironmentAfterTreeUnavailable(i, evidence)
}

func setCIEnvironmentAfterTreeUnavailable(i *CIEnvironmentInventory, evidence Evidence) {
	i.ProfileDeclared = Unavailable[bool](evidence)
	i.DeclaredApplicability = Unavailable[string](evidence)
	i.FlakePresent = Unavailable[bool](evidence)
	i.FlakeLockPresent = Unavailable[bool](evidence)
	i.WorkflowsState = StateUnavailable
}
