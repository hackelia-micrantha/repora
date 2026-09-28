package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"repoctl/internal/config"
	"repoctl/internal/posture"
)

const (
	DiscoveryKind    = "repora.bootstrap-discovery"
	DiscoveryVersion = 1
)

type DiscoveryObservation struct {
	Key      string   `json:"key"`
	State    string   `json:"state"`
	Value    *bool    `json:"value,omitempty"`
	Evidence []string `json:"evidence"`
}

type BaselineArtifact struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Source string `json:"source"`
}

type Discovery struct {
	Kind              string                 `json:"kind"`
	Version           int                    `json:"version"`
	Contract          ContractRef            `json:"contract"`
	Repository        string                 `json:"repository"`
	Registration      string                 `json:"registration"`
	DefaultBranch     string                 `json:"default_branch"`
	CommitSHA         string                 `json:"commit_sha"`
	TreeSHA           string                 `json:"tree_sha"`
	ManifestState     string                 `json:"manifest_state"`
	ManifestSHA256    string                 `json:"manifest_sha256,omitempty"`
	CIApplicability   string                 `json:"ci_applicability"`
	Observations      []DiscoveryObservation `json:"observations"`
	InheritedBaseline []BaselineArtifact     `json:"inherited_baseline"`
	FingerprintSHA256 string                 `json:"fingerprint_sha256"`
	ScanState         string                 `json:"scan_state"`
}

func Discover(ctx context.Context, reader posture.GitHubReader, spec config.Spec, fullName string, previous *Discovery) (Discovery, error) {
	repository, repoObs, err := reader.Repository(ctx, fullName)
	if err != nil {
		return Discovery{}, err
	}
	if !repoObs.Available {
		return Discovery{}, fmt.Errorf("GitHub repository %s is unavailable under current access", fullName)
	}
	branch, branchObs, err := reader.Branch(ctx, fullName, repository.DefaultBranch)
	if err != nil {
		return Discovery{}, err
	}
	if !branchObs.Available {
		return Discovery{}, fmt.Errorf("GitHub default branch %s#%s is unavailable under current access", fullName, repository.DefaultBranch)
	}
	tree, treeObs, err := reader.Tree(ctx, fullName, branch.TreeSHA)
	if err != nil {
		return Discovery{}, err
	}
	if !treeObs.Available {
		return Discovery{}, fmt.Errorf("GitHub tree %s@%s is unavailable under current access", fullName, branch.TreeSHA)
	}

	entries := make(map[string]posture.GitHubTreeEntry, len(tree.Entries))
	for _, entry := range tree.Entries {
		entries[entry.Path] = entry
	}

	baselineEntries := map[string]posture.GitHubTreeEntry{}
	baselineAvailable := false
	if strings.HasPrefix(fullName, "hackelia-micrantha/") && fullName != "hackelia-micrantha/.github" {
		baselineRepo, baselineRepoObs, baselineErr := reader.Repository(ctx, "hackelia-micrantha/.github")
		if baselineErr == nil && baselineRepoObs.Available {
			baselineBranch, baselineBranchObs, branchErr := reader.Branch(ctx, "hackelia-micrantha/.github", baselineRepo.DefaultBranch)
			if branchErr == nil && baselineBranchObs.Available {
				baselineTree, baselineTreeObs, treeErr := reader.Tree(ctx, "hackelia-micrantha/.github", baselineBranch.TreeSHA)
				if treeErr == nil && baselineTreeObs.Available && !baselineTree.Truncated {
					baselineAvailable = true
					for _, entry := range baselineTree.Entries {
						baselineEntries[entry.Path] = entry
					}
				}
			}
		}
	}

	discovery := Discovery{
		Kind:              DiscoveryKind,
		Version:           DiscoveryVersion,
		Contract:          Contract(),
		Repository:        fullName,
		Registration:      registrationState(spec, fullName),
		DefaultBranch:     repository.DefaultBranch,
		CommitSHA:         branch.CommitSHA,
		TreeSHA:           branch.TreeSHA,
		ManifestState:     "absent",
		CIApplicability:   "unresolved",
		Observations:      discoveryObservations(entries, tree.Truncated),
		InheritedBaseline: inheritedBaseline(fullName, entries, tree.Truncated, baselineEntries, baselineAvailable),
	}

	if entry, ok := entries[DefaultManifestPath]; ok && entry.Type == "blob" {
		data, obs, err := reader.Blob(ctx, fullName, entry.SHA)
		if err != nil {
			return Discovery{}, err
		}
		if !obs.Available {
			discovery.ManifestState = "unavailable"
		} else {
			discovery.ManifestSHA256 = digest(data)
			manifest, parseErr := ParseManifest(data)
			if parseErr != nil {
				discovery.ManifestState = "invalid"
			} else {
				discovery.ManifestState = "valid"
				discovery.CIApplicability = CIAplicability(manifest)
			}
		}
	}

	discovery.FingerprintSHA256 = discoveryFingerprint(discovery)
	discovery.ScanState = "first-discovery"
	if previous != nil {
		if previous.FingerprintSHA256 != discoveryFingerprint(*previous) {
			return Discovery{}, fmt.Errorf("previous discovery fingerprint does not match its evidence")
		}
		if previous.Repository != fullName {
			return Discovery{}, fmt.Errorf("previous discovery repository %q does not match %q", previous.Repository, fullName)
		}
		if previous.FingerprintSHA256 == discovery.FingerprintSHA256 {
			discovery.ScanState = "unchanged"
		} else {
			discovery.ScanState = "changed"
		}
	}
	return discovery, discovery.Validate()
}

func registrationState(spec config.Spec, fullName string) string {
	for _, repo := range spec.Repos {
		endpoints := append([]config.Endpoint{repo.Canonical}, repo.Mirrors...)
		for _, endpoint := range endpoints {
			if endpoint.Provider != "github" {
				continue
			}
			path, err := endpoint.RepositoryPath()
			if err == nil && strings.EqualFold(path, fullName) {
				return "registered"
			}
		}
	}
	return "unregistered"
}

func discoveryObservations(entries map[string]posture.GitHubTreeEntry, truncated bool) []DiscoveryObservation {
	present := func(key string, candidates ...string) DiscoveryObservation {
		for _, candidate := range candidates {
			if entry, ok := entries[candidate]; ok && entry.Type == "blob" {
				value := true
				return DiscoveryObservation{Key: key, State: "observed", Value: &value, Evidence: []string{candidate}}
			}
		}
		if truncated {
			return DiscoveryObservation{Key: key, State: "unknown", Evidence: []string{"github.git_tree:truncated"}}
		}
		value := false
		return DiscoveryObservation{Key: key, State: "observed", Value: &value, Evidence: []string{"github.git_tree"}}
	}

	workflows := false
	workflowEvidence := []string{}
	for path, entry := range entries {
		if entry.Type != "blob" || !strings.HasPrefix(path, ".github/workflows/") {
			continue
		}
		lower := strings.ToLower(path)
		if strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml") {
			workflows = true
			workflowEvidence = append(workflowEvidence, path)
		}
	}
	sort.Strings(workflowEvidence)
	workflowObservation := DiscoveryObservation{Key: "workflows_present", State: "observed", Value: &workflows, Evidence: workflowEvidence}
	if !workflows && truncated {
		workflowObservation = DiscoveryObservation{Key: "workflows_present", State: "unknown", Evidence: []string{"github.git_tree:truncated"}}
	}
	if !workflows && !truncated {
		workflowObservation.Evidence = []string{"github.git_tree"}
	}

	out := []DiscoveryObservation{
		present("bootstrap_manifest_present", DefaultManifestPath),
		present("flake_present", "flake.nix"),
		present("flake_lock_present", "flake.lock"),
		present("license_present", "LICENSE", "LICENSE.md", "LICENSE.txt", "COPYING", "COPYING.md", "COPYING.txt"),
		workflowObservation,
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func inheritedBaseline(fullName string, entries map[string]posture.GitHubTreeEntry, truncated bool, sourceEntries map[string]posture.GitHubTreeEntry, sourceAvailable bool) []BaselineArtifact {
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 || parts[0] != "hackelia-micrantha" || parts[1] == ".github" {
		return []BaselineArtifact{}
	}
	definitions := []struct {
		name       string
		candidates []string
	}{
		{"security-guidance", []string{"SECURITY.md", ".github/SECURITY.md", "docs/SECURITY.md"}},
		{"contributing-guidance", []string{"CONTRIBUTING.md", ".github/CONTRIBUTING.md", "docs/CONTRIBUTING.md"}},
		{"support-guidance", []string{"SUPPORT.md", ".github/SUPPORT.md", "docs/SUPPORT.md"}},
		{"code-of-conduct", []string{"CODE_OF_CONDUCT.md", ".github/CODE_OF_CONDUCT.md", "docs/CODE_OF_CONDUCT.md"}},
		{"pull-request-template", []string{"PULL_REQUEST_TEMPLATE.md", ".github/PULL_REQUEST_TEMPLATE.md", "docs/PULL_REQUEST_TEMPLATE.md"}},
	}
	out := make([]BaselineArtifact, 0, len(definitions)+1)
	for _, definition := range definitions {
		state := "unknown"
		for _, candidate := range definition.candidates {
			if entry, ok := entries[candidate]; ok && entry.Type == "blob" {
				state = "local-override"
				break
			}
		}
		if state != "local-override" && !truncated && sourceAvailable {
			for _, candidate := range definition.candidates {
				if entry, ok := sourceEntries[candidate]; ok && entry.Type == "blob" {
					state = "provider-inherited"
					break
				}
			}
		}
		out = append(out, BaselineArtifact{Name: definition.name, State: state, Source: "hackelia-micrantha/.github"})
	}
	issueState := "unknown"
	for path, entry := range entries {
		if entry.Type == "blob" && strings.HasPrefix(path, ".github/ISSUE_TEMPLATE/") {
			issueState = "local-override"
			break
		}
	}
	if issueState != "local-override" && !truncated && sourceAvailable {
		for path, entry := range sourceEntries {
			if entry.Type == "blob" && strings.HasPrefix(path, ".github/ISSUE_TEMPLATE/") {
				issueState = "provider-inherited"
				break
			}
		}
	}
	out = append(out, BaselineArtifact{Name: "issue-templates", State: issueState, Source: "hackelia-micrantha/.github"})
	return out
}

func discoveryFingerprint(discovery Discovery) string {
	copy := discovery
	copy.FingerprintSHA256 = ""
	copy.ScanState = ""
	data, _ := json.Marshal(copy)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func ParseDiscovery(data []byte) (Discovery, error) {
	var discovery Discovery
	if err := strictDecode(data, &discovery); err != nil {
		return Discovery{}, fmt.Errorf("decode bootstrap discovery: %w", err)
	}
	if err := discovery.Validate(); err != nil {
		return Discovery{}, err
	}
	return discovery, nil
}

func (d Discovery) Validate() error {
	if d.Kind != DiscoveryKind || d.Version != DiscoveryVersion {
		return fmt.Errorf("unsupported bootstrap discovery contract: kind=%q version=%d", d.Kind, d.Version)
	}
	if d.Contract != Contract() {
		return fmt.Errorf("bootstrap discovery contract reference does not match pinned contract")
	}
	if strings.TrimSpace(d.Repository) == "" || strings.Count(d.Repository, "/") != 1 {
		return fmt.Errorf("bootstrap discovery requires OWNER/REPO identity")
	}
	if d.Registration != "registered" && d.Registration != "unregistered" {
		return fmt.Errorf("invalid bootstrap registration state %q", d.Registration)
	}
	switch d.ManifestState {
	case "absent", "valid", "invalid", "unavailable":
	default:
		return fmt.Errorf("invalid manifest_state %q", d.ManifestState)
	}
	switch d.CIApplicability {
	case "applicable", "not-applicable", "unresolved":
	default:
		return fmt.Errorf("invalid ci_applicability %q", d.CIApplicability)
	}
	if d.Observations == nil || d.InheritedBaseline == nil || !validDigest(d.FingerprintSHA256) {
		return fmt.Errorf("bootstrap discovery requires arrays and fingerprint")
	}
	switch d.ScanState {
	case "first-discovery", "unchanged", "changed":
	default:
		return fmt.Errorf("invalid scan_state %q", d.ScanState)
	}
	return nil
}

func (d Discovery) Marshal() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode bootstrap discovery: %w", err)
	}
	return append(data, '\n'), nil
}
