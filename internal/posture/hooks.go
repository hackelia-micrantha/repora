package posture

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	HooksInventoryKind    = "repora.posture-hooks"
	HooksInventoryVersion = 1
	HooksProfileKind      = "repora.posture-hooks-profile"
	HooksProfileVersion   = 1
	hooksProfilePath      = ".repora/posture-hooks.yaml"
	maxHooksBytes         = 1 << 20
	maxHookTargets        = 128
)

type HooksProfile struct {
	Kind           string   `json:"kind" yaml:"kind"`
	Version        int      `json:"version" yaml:"version"`
	Manager        string   `json:"manager" yaml:"manager"`
	HookPaths      []string `json:"hook_paths" yaml:"hook_paths"`
	RequiredChecks []string `json:"required_checks" yaml:"required_checks"`
	BootstrapDocs  []string `json:"bootstrap_docs" yaml:"bootstrap_docs"`
	BypassDocs     []string `json:"bypass_docs" yaml:"bypass_docs"`
}

type HookEntrypointFact struct {
	Path          string     `json:"path"`
	Configured    Fact[bool] `json:"configured"`
	Executable    Fact[bool] `json:"executable"`
	NetworkLoaded Fact[bool] `json:"network_loaded"`
}

type LocalCheckFact struct {
	Name       string     `json:"name"`
	Configured Fact[bool] `json:"configured"`
	CICovered  Fact[bool] `json:"ci_covered"`
}

type HooksInventory struct {
	Kind             string               `json:"kind"`
	Version          int                  `json:"version"`
	Repository       RepositoryIdentity   `json:"repository"`
	DefaultBranch    Fact[string]         `json:"default_branch"`
	DefaultCommit    Fact[string]         `json:"default_commit"`
	ProfileDeclared  Fact[bool]           `json:"profile_declared"`
	Manager          Fact[string]         `json:"manager"`
	Entrypoints      []HookEntrypointFact `json:"entrypoints"`
	RequiredChecks   []LocalCheckFact     `json:"required_checks"`
	BootstrapPresent Fact[bool]           `json:"bootstrap_instructions_present"`
	BypassPresent    Fact[bool]           `json:"bypass_documentation_present"`
	Evidence         []Evidence           `json:"evidence"`
}

func ParseHooksProfile(data []byte) (HooksProfile, error) {
	if len(data) > maxHooksBytes {
		return HooksProfile{}, fmt.Errorf("hooks profile exceeds %d bytes", maxHooksBytes)
	}
	var profile HooksProfile
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&profile); err != nil {
		return HooksProfile{}, fmt.Errorf("parse hooks profile: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return HooksProfile{}, fmt.Errorf("hooks profile must contain exactly one YAML document")
		}
		return HooksProfile{}, fmt.Errorf("parse hooks profile trailing content: %w", err)
	}
	if err := profile.Validate(); err != nil {
		return HooksProfile{}, err
	}
	profile.HookPaths = sortedUnique(profile.HookPaths)
	profile.RequiredChecks = sortedUnique(profile.RequiredChecks)
	profile.BootstrapDocs = sortedUnique(profile.BootstrapDocs)
	profile.BypassDocs = sortedUnique(profile.BypassDocs)
	return profile, nil
}

func (p HooksProfile) Validate() error {
	if p.Kind != HooksProfileKind || p.Version != HooksProfileVersion {
		return fmt.Errorf("unsupported hooks profile contract: kind=%q version=%d", p.Kind, p.Version)
	}
	if len(p.HookPaths)+len(p.RequiredChecks)+len(p.BootstrapDocs)+len(p.BypassDocs) > maxHookTargets {
		return fmt.Errorf("hooks profile exceeds %d observation targets", maxHookTargets)
	}
	for _, value := range append(append(append([]string{}, p.HookPaths...), p.BootstrapDocs...), p.BypassDocs...) {
		if err := validateDocumentationPath(value); err != nil {
			return err
		}
	}
	for _, check := range p.RequiredChecks {
		if strings.TrimSpace(check) == "" {
			return fmt.Errorf("required check names must not be empty")
		}
	}
	return nil
}

func newHooksInventory(fullName string) HooksInventory {
	return HooksInventory{Kind: HooksInventoryKind, Version: HooksInventoryVersion, Repository: RepositoryIdentity{Provider: "github", FullName: fullName}, Entrypoints: []HookEntrypointFact{}, RequiredChecks: []LocalCheckFact{}, Evidence: []Evidence{}}
}

func (i HooksInventory) Validate() error {
	if i.Kind != HooksInventoryKind || i.Version != HooksInventoryVersion {
		return fmt.Errorf("unsupported hooks inventory contract: kind=%q version=%d", i.Kind, i.Version)
	}
	if i.Repository.Provider != "github" {
		return fmt.Errorf("hooks inventory provider must be github")
	}
	if _, _, err := splitGitHubFullName(i.Repository.FullName); err != nil {
		return err
	}
	checks := []error{validateFact("default_branch", i.DefaultBranch), validateFact("default_commit", i.DefaultCommit), validateFact("profile_declared", i.ProfileDeclared), validateFact("manager", i.Manager), validateFact("bootstrap_instructions_present", i.BootstrapPresent), validateFact("bypass_documentation_present", i.BypassPresent)}
	for _, err := range checks {
		if err != nil {
			return err
		}
	}
	if i.Entrypoints == nil || i.RequiredChecks == nil || i.Evidence == nil {
		return fmt.Errorf("hooks inventory arrays are required")
	}
	for idx, hook := range i.Entrypoints {
		if err := validateDocumentationPath(hook.Path); err != nil {
			return fmt.Errorf("entrypoint[%d]: %w", idx, err)
		}
		if err := validateFact("configured", hook.Configured); err != nil {
			return err
		}
		if err := validateFact("executable", hook.Executable); err != nil {
			return err
		}
		if err := validateFact("network_loaded", hook.NetworkLoaded); err != nil {
			return err
		}
	}
	for idx, check := range i.RequiredChecks {
		if strings.TrimSpace(check.Name) == "" {
			return fmt.Errorf("required_checks[%d] name is required", idx)
		}
		if err := validateFact("configured", check.Configured); err != nil {
			return err
		}
		if err := validateFact("ci_covered", check.CICovered); err != nil {
			return err
		}
	}
	return nil
}

func (i HooksInventory) Marshal() ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode hooks posture inventory: %w", err)
	}
	return append(data, '\n'), nil
}

func CollectGitHubHooks(ctx context.Context, reader GitHubReader, fullName string) (HooksInventory, error) {
	if _, _, err := splitGitHubFullName(fullName); err != nil {
		return HooksInventory{}, err
	}
	inventory := newHooksInventory(fullName)
	repo, repoObs, err := reader.Repository(ctx, fullName)
	if err != nil {
		return HooksInventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, repoObs.Evidence)
	if !repoObs.Available {
		setHooksUnavailable(&inventory, repoObs.Evidence)
		return inventory, inventory.Validate()
	}
	inventory.DefaultBranch = Observed(repo.DefaultBranch, repoObs.Evidence)
	branch, branchObs, err := reader.Branch(ctx, fullName, repo.DefaultBranch)
	if err != nil {
		return HooksInventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, branchObs.Evidence)
	if !branchObs.Available {
		setHooksAfterBranchUnavailable(&inventory, branchObs.Evidence)
		return inventory, inventory.Validate()
	}
	inventory.DefaultCommit = Observed(branch.CommitSHA, branchObs.Evidence)
	tree, treeObs, err := reader.Tree(ctx, fullName, branch.TreeSHA)
	if err != nil {
		return HooksInventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, treeObs.Evidence)
	if !treeObs.Available {
		setHooksAfterTreeUnavailable(&inventory, treeObs.Evidence)
		return inventory, inventory.Validate()
	}
	entries := map[string]GitHubTreeEntry{}
	for _, entry := range tree.Entries {
		entries[entry.Path] = entry
	}
	profile, profileState, profileEvidence, err := loadHooksProfile(ctx, reader, fullName, tree, entries, treeObs.Evidence)
	if err != nil {
		return HooksInventory{}, err
	}
	inventory.ProfileDeclared = presenceFact(entries, tree, hooksProfilePath, treeObs.Evidence)
	if profileState != StateObserved {
		inventory.Manager = factForState[string](profileState, profileEvidence)
		inventory.BootstrapPresent = factForState[bool](profileState, profileEvidence)
		inventory.BypassPresent = factForState[bool](profileState, profileEvidence)
		return inventory, inventory.Validate()
	}
	manager, candidates := detectHookManager(entries)
	if profile.Manager != "" {
		manager = profile.Manager
	}
	candidates = sortedUnique(append(candidates, profile.HookPaths...))
	switch {
	case manager != "":
		inventory.Manager = Observed(manager, treeObs.Evidence)
	case tree.Truncated:
		inventory.Manager = Unknown[string](evidenceWithDetail(treeObs.Evidence, "Git tree is truncated; absence of a hook manager cannot be established"))
	default:
		inventory.Manager = Observed("none", treeObs.Evidence)
	}
	for _, hookPath := range candidates {
		entry, ok := entries[hookPath]
		configured := presenceFact(entries, tree, hookPath, treeObs.Evidence)
		executable := Unknown[bool](evidenceWithDetail(treeObs.Evidence, "GitHub tree mode is not exposed by this posture reader"))
		networkLoaded := Unknown[bool](treeObs.Evidence)
		if ok && entry.Type == "blob" {
			data, obs, err := reader.Blob(ctx, fullName, entry.SHA)
			if err != nil {
				return HooksInventory{}, err
			}
			if !obs.Available {
				networkLoaded = Unavailable[bool](obs.Evidence)
			} else if len(data) > maxHooksBytes {
				networkLoaded = Unknown[bool](evidenceWithDetail(obs.Evidence, "hook content exceeds bounded static-inspection limit"))
			} else {
				networkLoaded = Observed(containsNetworkLoader(data), obs.Evidence)
			}
		}
		inventory.Entrypoints = append(inventory.Entrypoints, HookEntrypointFact{Path: hookPath, Configured: configured, Executable: executable, NetworkLoaded: networkLoaded})
	}
	if len(profile.RequiredChecks) > 0 {
		workflowData, workflowState, workflowEvidence, err := collectWorkflowText(ctx, reader, fullName, tree, entries)
		if err != nil {
			return HooksInventory{}, err
		}
		for _, check := range profile.RequiredChecks {
			covered := factForState[bool](workflowState, workflowEvidence)
			matched := strings.Contains(strings.ToLower(workflowData), strings.ToLower(check))
			if matched {
				covered = Observed(true, workflowEvidence)
			} else if workflowState == StateObserved {
				covered = Observed(false, workflowEvidence)
			}
			inventory.RequiredChecks = append(inventory.RequiredChecks, LocalCheckFact{Name: check, Configured: Observed(true, profileEvidence), CICovered: covered})
		}
	}
	inventory.BootstrapPresent = anyPathPresent(profile.BootstrapDocs, entries, tree, treeObs.Evidence)
	inventory.BypassPresent = anyPathPresent(profile.BypassDocs, entries, tree, treeObs.Evidence)
	return inventory, inventory.Validate()
}

func loadHooksProfile(ctx context.Context, reader GitHubReader, fullName string, tree GitHubTree, entries map[string]GitHubTreeEntry, treeEvidence Evidence) (HooksProfile, FactState, Evidence, error) {
	entry, ok := entries[hooksProfilePath]
	if !ok {
		if tree.Truncated {
			evidence := evidenceWithDetail(treeEvidence, "Git tree is truncated; hooks profile presence is unknown")
			return HooksProfile{}, StateUnknown, evidence, nil
		}
		return HooksProfile{}, StateObserved, Evidence{Source: "repora.builtin", Reference: "hooks-profile:baseline"}, nil
	}
	if entry.Type != "blob" {
		evidence := evidenceWithDetail(treeEvidence, "hooks profile exists but is not a blob")
		return HooksProfile{}, StateUnknown, evidence, nil
	}
	data, obs, err := reader.Blob(ctx, fullName, entry.SHA)
	if err != nil {
		return HooksProfile{}, "", Evidence{}, err
	}
	if !obs.Available {
		return HooksProfile{}, StateUnavailable, obs.Evidence, nil
	}
	profile, err := ParseHooksProfile(data)
	if err != nil {
		return HooksProfile{}, StateUnknown, evidenceWithDetail(obs.Evidence, "declared hooks profile is malformed or unsupported"), nil
	}
	return profile, StateObserved, obs.Evidence, nil
}

func detectHookManager(entries map[string]GitHubTreeEntry) (string, []string) {
	if _, ok := entries[".pre-commit-config.yaml"]; ok {
		return "pre-commit", []string{".pre-commit-config.yaml"}
	}
	if _, ok := entries["lefthook.yml"]; ok {
		return "lefthook", []string{"lefthook.yml"}
	}
	if _, ok := entries["lefthook.yaml"]; ok {
		return "lefthook", []string{"lefthook.yaml"}
	}
	var husky []string
	var custom []string
	for p, entry := range entries {
		if entry.Type != "blob" {
			continue
		}
		if strings.HasPrefix(p, ".husky/") && !strings.HasPrefix(p, ".husky/_/") {
			husky = append(husky, p)
		}
		if strings.HasPrefix(p, ".githooks/") {
			custom = append(custom, p)
		}
	}
	if len(husky) > 0 {
		return "husky", sortedUnique(husky)
	}
	if len(custom) > 0 {
		return "custom", sortedUnique(custom)
	}
	return "", []string{}
}

func containsNetworkLoader(data []byte) bool {
	text := strings.ToLower(string(data))
	return strings.Contains(text, "curl ") || strings.Contains(text, "wget ") || strings.Contains(text, "http://") || strings.Contains(text, "https://")
}

func presenceFact(entries map[string]GitHubTreeEntry, tree GitHubTree, target string, evidence Evidence) Fact[bool] {
	if _, ok := entries[target]; ok {
		return Observed(true, evidence)
	}
	if tree.Truncated {
		return Unknown[bool](evidenceWithDetail(evidence, "Git tree is truncated; path absence cannot be established"))
	}
	return Observed(false, evidence)
}

func anyPathPresent(paths []string, entries map[string]GitHubTreeEntry, tree GitHubTree, evidence Evidence) Fact[bool] {
	if len(paths) == 0 {
		return Unknown[bool](evidenceWithDetail(evidence, "no documentation expectation declared"))
	}
	for _, p := range paths {
		if _, ok := entries[p]; ok {
			return Observed(true, evidence)
		}
	}
	if tree.Truncated {
		return Unknown[bool](evidenceWithDetail(evidence, "Git tree is truncated; documentation absence cannot be established"))
	}
	return Observed(false, evidence)
}

func collectWorkflowText(ctx context.Context, reader GitHubReader, fullName string, tree GitHubTree, entries map[string]GitHubTreeEntry) (string, FactState, Evidence, error) {
	paths := []string{}
	for p, entry := range entries {
		if entry.Type == "blob" && strings.HasPrefix(p, ".github/workflows/") && (strings.HasSuffix(p, ".yml") || strings.HasSuffix(p, ".yaml")) {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		if tree.Truncated {
			ev := Evidence{Source: "github.git_tree", Reference: fullName, Detail: "tree truncated; workflow coverage unknown"}
			return "", StateUnknown, ev, nil
		}
		return "", StateObserved, Evidence{Source: "github.git_tree", Reference: fullName, Detail: "no GitHub Actions workflows observed"}, nil
	}
	var builder strings.Builder
	var evidence Evidence
	for _, p := range paths {
		data, obs, err := reader.Blob(ctx, fullName, entries[p].SHA)
		if err != nil {
			return "", "", Evidence{}, err
		}
		if !obs.Available {
			return "", StateUnavailable, obs.Evidence, nil
		}
		if len(data) <= maxHooksBytes {
			builder.Write(data)
			builder.WriteByte('\n')
		}
		evidence = obs.Evidence
	}
	if tree.Truncated {
		return builder.String(), StateUnknown, evidenceWithDetail(evidence, "Git tree is truncated; workflow set may be incomplete"), nil
	}
	return builder.String(), StateObserved, evidence, nil
}

func setHooksUnavailable(i *HooksInventory, evidence Evidence) {
	i.DefaultBranch = Unavailable[string](evidence)
	setHooksAfterBranchUnavailable(i, evidence)
}

func setHooksAfterBranchUnavailable(i *HooksInventory, evidence Evidence) {
	i.DefaultCommit = Unavailable[string](evidence)
	setHooksAfterTreeUnavailable(i, evidence)
}

func setHooksAfterTreeUnavailable(i *HooksInventory, evidence Evidence) {
	i.ProfileDeclared = Unavailable[bool](evidence)
	i.Manager = Unavailable[string](evidence)
	i.BootstrapPresent = Unavailable[bool](evidence)
	i.BypassPresent = Unavailable[bool](evidence)
}
