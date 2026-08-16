package posture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DocumentationInventoryKind    = "repora.posture-documentation"
	DocumentationInventoryVersion = 1
	DocumentationProfileKind      = "repora.posture-documentation-profile"
	DocumentationProfileVersion   = 1

	documentationProfilePath = ".repora/posture-documentation.yaml"
	documentRouterPath       = ".repora/document-router.yaml"
	maxDocumentationBytes   = 2 << 20
	maxDocumentationTargets = 256
)

var inlineMarkdownLink = regexp.MustCompile(`\[[^\]]*\]\(\s*<?([^\s)>]+)>?(?:\s+[^)]*)?\)`)

type DocumentationProfile struct {
	Kind           string                       `json:"kind" yaml:"kind"`
	Version        int                          `json:"version" yaml:"version"`
	Name           string                       `json:"name" yaml:"name"`
	Documents      []string                     `json:"documents" yaml:"documents"`
	README         DocumentationREADMEProfile   `json:"readme" yaml:"readme"`
	ContentMarkers []DocumentationContentMarker `json:"content_markers" yaml:"content_markers"`
}

type DocumentationREADMEProfile struct {
	Path     string   `json:"path" yaml:"path"`
	Sections []string `json:"sections" yaml:"sections"`
	Links    []string `json:"links" yaml:"links"`
}

type DocumentationContentMarker struct {
	ID       string `json:"id" yaml:"id"`
	Path     string `json:"path" yaml:"path"`
	Contains string `json:"contains" yaml:"contains"`
}

type DocumentationDocumentFact struct {
	Path      string       `json:"path"`
	Present   Fact[bool]   `json:"present"`
	TrustTier Fact[string] `json:"trust_tier"`
}

type DocumentationSectionFact struct {
	Section string     `json:"section"`
	Present Fact[bool] `json:"present"`
}

type DocumentationLinkFact struct {
	Target  string     `json:"target"`
	Present Fact[bool] `json:"present"`
}

type DocumentationMarkerFact struct {
	ID             string     `json:"id"`
	Path           string     `json:"path"`
	ExpectedSHA256 string     `json:"expected_sha256"`
	Present        Fact[bool] `json:"present"`
}

type DocumentationInventory struct {
	Kind                   string                    `json:"kind"`
	Version                int                       `json:"version"`
	Repository             RepositoryIdentity        `json:"repository"`
	DefaultBranch          Fact[string]              `json:"default_branch"`
	DefaultCommit          Fact[string]              `json:"default_commit"`
	ProfileDeclared        Fact[bool]                `json:"profile_declared"`
	ProfileName            Fact[string]              `json:"profile_name"`
	READMEPath             string                    `json:"readme_path"`
	READMEPresent          Fact[bool]                `json:"readme_present"`
	Documents              []DocumentationDocumentFact `json:"documents"`
	READMESections         []DocumentationSectionFact  `json:"readme_sections"`
	READMELinks            []DocumentationLinkFact     `json:"readme_links"`
	ContentMarkers         []DocumentationMarkerFact   `json:"content_markers"`
	RoutingMetadataPresent Fact[bool]                `json:"routing_metadata_present"`
	RoutingMetadataValid   Fact[bool]                `json:"routing_metadata_valid"`
	Evidence               []Evidence                `json:"evidence"`
}

func DefaultDocumentationProfile() DocumentationProfile {
	return DocumentationProfile{
		Kind:      DocumentationProfileKind,
		Version:   DocumentationProfileVersion,
		Name:      "baseline",
		Documents: []string{"README.md"},
		README: DocumentationREADMEProfile{
			Path:     "README.md",
			Sections: []string{},
			Links:    []string{},
		},
		ContentMarkers: []DocumentationContentMarker{},
	}
}

func ParseDocumentationProfile(data []byte) (DocumentationProfile, error) {
	if len(data) > maxDocumentationBytes {
		return DocumentationProfile{}, fmt.Errorf("documentation profile exceeds %d bytes", maxDocumentationBytes)
	}
	var profile DocumentationProfile
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&profile); err != nil {
		return DocumentationProfile{}, fmt.Errorf("parse documentation profile: %w", err)
	}
	if err := profile.Validate(); err != nil {
		return DocumentationProfile{}, err
	}
	profile.Documents = sortedUnique(profile.Documents)
	profile.README.Sections = sortedUnique(profile.README.Sections)
	profile.README.Links = sortedUnique(profile.README.Links)
	sort.Slice(profile.ContentMarkers, func(i, j int) bool {
		if profile.ContentMarkers[i].ID == profile.ContentMarkers[j].ID {
			return profile.ContentMarkers[i].Path < profile.ContentMarkers[j].Path
		}
		return profile.ContentMarkers[i].ID < profile.ContentMarkers[j].ID
	})
	return profile, nil
}

func (p DocumentationProfile) Validate() error {
	if p.Kind != DocumentationProfileKind || p.Version != DocumentationProfileVersion {
		return fmt.Errorf("unsupported documentation profile contract: kind=%q version=%d", p.Kind, p.Version)
	}
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("documentation profile name is required")
	}
	if len(p.Documents)+len(p.README.Sections)+len(p.README.Links)+len(p.ContentMarkers) > maxDocumentationTargets {
		return fmt.Errorf("documentation profile exceeds %d observation targets", maxDocumentationTargets)
	}
	if err := validateDocumentationPath(p.README.Path); err != nil {
		return fmt.Errorf("readme path: %w", err)
	}
	seenDocuments := map[string]struct{}{}
	for _, document := range p.Documents {
		if err := validateDocumentationPath(document); err != nil {
			return fmt.Errorf("document %q: %w", document, err)
		}
		if _, exists := seenDocuments[document]; exists {
			return fmt.Errorf("duplicate document path %q", document)
		}
		seenDocuments[document] = struct{}{}
	}
	seenSections := map[string]struct{}{}
	for _, section := range p.README.Sections {
		normalized := normalizeHeading(section)
		if normalized == "" {
			return fmt.Errorf("README section names must not be empty")
		}
		if _, exists := seenSections[normalized]; exists {
			return fmt.Errorf("duplicate README section %q", section)
		}
		seenSections[normalized] = struct{}{}
	}
	seenLinks := map[string]struct{}{}
	for _, target := range p.README.Links {
		if err := validateDocumentationPath(target); err != nil {
			return fmt.Errorf("README link target %q: %w", target, err)
		}
		if _, exists := seenLinks[target]; exists {
			return fmt.Errorf("duplicate README link target %q", target)
		}
		seenLinks[target] = struct{}{}
	}
	seenMarkers := map[string]struct{}{}
	for _, marker := range p.ContentMarkers {
		if strings.TrimSpace(marker.ID) == "" {
			return fmt.Errorf("content marker id is required")
		}
		if _, exists := seenMarkers[marker.ID]; exists {
			return fmt.Errorf("duplicate content marker id %q", marker.ID)
		}
		seenMarkers[marker.ID] = struct{}{}
		if err := validateDocumentationPath(marker.Path); err != nil {
			return fmt.Errorf("content marker %q path: %w", marker.ID, err)
		}
		if marker.Contains == "" {
			return fmt.Errorf("content marker %q contains value is required", marker.ID)
		}
		if len(marker.Contains) > 4096 {
			return fmt.Errorf("content marker %q exceeds 4096 bytes", marker.ID)
		}
	}
	return nil
}

func validateDocumentationPath(value string) error {
	if value == "" {
		return fmt.Errorf("path is required")
	}
	if strings.Contains(value, "\\") || strings.HasPrefix(value, "/") {
		return fmt.Errorf("path must be repository-relative and use '/' separators")
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != value {
		return fmt.Errorf("path must be normalized and contained within the repository")
	}
	return nil
}

func newDocumentationInventory(fullName string) DocumentationInventory {
	return DocumentationInventory{
		Kind:           DocumentationInventoryKind,
		Version:        DocumentationInventoryVersion,
		Repository:     RepositoryIdentity{Provider: "github", FullName: fullName},
		Documents:      []DocumentationDocumentFact{},
		READMESections: []DocumentationSectionFact{},
		READMELinks:    []DocumentationLinkFact{},
		ContentMarkers: []DocumentationMarkerFact{},
		Evidence:       []Evidence{},
	}
}

func (i DocumentationInventory) Validate() error {
	if i.Kind != DocumentationInventoryKind || i.Version != DocumentationInventoryVersion {
		return fmt.Errorf("unsupported documentation inventory contract: kind=%q version=%d", i.Kind, i.Version)
	}
	if i.Repository.Provider != "github" {
		return fmt.Errorf("documentation inventory provider must be github")
	}
	if _, _, err := splitGitHubFullName(i.Repository.FullName); err != nil {
		return err
	}
	for name, err := range map[string]error{
		"default_branch":           validateFact("default_branch", i.DefaultBranch),
		"default_commit":           validateFact("default_commit", i.DefaultCommit),
		"profile_declared":         validateFact("profile_declared", i.ProfileDeclared),
		"profile_name":             validateFact("profile_name", i.ProfileName),
		"readme_present":           validateFact("readme_present", i.READMEPresent),
		"routing_metadata_present": validateFact("routing_metadata_present", i.RoutingMetadataPresent),
		"routing_metadata_valid":   validateFact("routing_metadata_valid", i.RoutingMetadataValid),
	} {
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if i.Documents == nil || i.READMESections == nil || i.READMELinks == nil || i.ContentMarkers == nil || i.Evidence == nil {
		return fmt.Errorf("documentation inventory arrays are required")
	}
	if i.READMEPath != "" {
		if err := validateDocumentationPath(i.READMEPath); err != nil {
			return fmt.Errorf("readme_path: %w", err)
		}
	}
	for idx, document := range i.Documents {
		if err := validateDocumentationPath(document.Path); err != nil {
			return fmt.Errorf("document[%d] path: %w", idx, err)
		}
		if err := validateFact("document present", document.Present); err != nil {
			return fmt.Errorf("document[%d]: %w", idx, err)
		}
		if err := validateFact("document trust tier", document.TrustTier); err != nil {
			return fmt.Errorf("document[%d]: %w", idx, err)
		}
	}
	for idx, section := range i.READMESections {
		if strings.TrimSpace(section.Section) == "" {
			return fmt.Errorf("readme_sections[%d] section is required", idx)
		}
		if err := validateFact("README section", section.Present); err != nil {
			return fmt.Errorf("readme_sections[%d]: %w", idx, err)
		}
	}
	for idx, link := range i.READMELinks {
		if err := validateDocumentationPath(link.Target); err != nil {
			return fmt.Errorf("readme_links[%d] target: %w", idx, err)
		}
		if err := validateFact("README link", link.Present); err != nil {
			return fmt.Errorf("readme_links[%d]: %w", idx, err)
		}
	}
	for idx, marker := range i.ContentMarkers {
		if strings.TrimSpace(marker.ID) == "" || len(marker.ExpectedSHA256) != 64 {
			return fmt.Errorf("content_markers[%d] id and SHA-256 digest are required", idx)
		}
		if err := validateDocumentationPath(marker.Path); err != nil {
			return fmt.Errorf("content_markers[%d] path: %w", idx, err)
		}
		if err := validateFact("content marker", marker.Present); err != nil {
			return fmt.Errorf("content_markers[%d]: %w", idx, err)
		}
	}
	return nil
}

func (i DocumentationInventory) Marshal() ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode documentation posture inventory: %w", err)
	}
	return append(data, '\n'), nil
}

type documentContent struct {
	State    FactState
	Data     []byte
	Evidence Evidence
}

type trustRule struct {
	Tier     string
	Patterns []string
	Index    int
}

type trustRouter struct {
	Rules []trustRule
}

func CollectGitHubDocumentation(ctx context.Context, reader GitHubReader, fullName string) (DocumentationInventory, error) {
	if _, _, err := splitGitHubFullName(fullName); err != nil {
		return DocumentationInventory{}, err
	}
	inventory := newDocumentationInventory(fullName)
	repository, repoObs, err := reader.Repository(ctx, fullName)
	if err != nil {
		return DocumentationInventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, repoObs.Evidence)
	if !repoObs.Available {
		setDocumentationUnavailable(&inventory, repoObs.Evidence)
		return inventory, inventory.Validate()
	}
	inventory.DefaultBranch = Observed(repository.DefaultBranch, repoObs.Evidence)

	branch, branchObs, err := reader.Branch(ctx, fullName, repository.DefaultBranch)
	if err != nil {
		return DocumentationInventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, branchObs.Evidence)
	if !branchObs.Available {
		setDocumentationAfterBranchUnavailable(&inventory, branchObs.Evidence)
		return inventory, inventory.Validate()
	}
	inventory.DefaultCommit = Observed(branch.CommitSHA, branchObs.Evidence)

	tree, treeObs, err := reader.Tree(ctx, fullName, branch.TreeSHA)
	if err != nil {
		return DocumentationInventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, treeObs.Evidence)
	if !treeObs.Available {
		setDocumentationAfterTreeUnavailable(&inventory, treeObs.Evidence)
		return inventory, inventory.Validate()
	}
	entries := make(map[string]GitHubTreeEntry, len(tree.Entries))
	for _, entry := range tree.Entries {
		entries[entry.Path] = entry
	}

	profile, profileEvidence, profileState, declared, err := loadDocumentationProfile(ctx, reader, fullName, tree, entries, treeObs.Evidence)
	if err != nil {
		return DocumentationInventory{}, err
	}
	inventory.ProfileDeclared = declared
	if profileState == StateObserved {
		inventory.ProfileName = Observed(profile.Name, profileEvidence)
		inventory.READMEPath = profile.README.Path
	} else if profileState == StateUnavailable {
		inventory.ProfileName = Unavailable[string](profileEvidence)
	} else {
		inventory.ProfileName = Unknown[string](profileEvidence)
	}

	router, routerState, routerEvidence, routerPresent, routerValid, err := loadTrustRouter(ctx, reader, fullName, tree, entries, treeObs.Evidence)
	if err != nil {
		return DocumentationInventory{}, err
	}
	inventory.RoutingMetadataPresent = routerPresent
	inventory.RoutingMetadataValid = routerValid
	if routerEvidence.Source != "" {
		inventory.Evidence = append(inventory.Evidence, routerEvidence)
	}

	if profileState != StateObserved {
		inventory.READMEPresent = factForState[bool](profileState, profileEvidence)
		return inventory, inventory.Validate()
	}

	contentCache := map[string]documentContent{}
	loadContent := func(documentPath string) (documentContent, error) {
		if cached, ok := contentCache[documentPath]; ok {
			return cached, nil
		}
		content, err := loadDocumentationContent(ctx, reader, fullName, documentPath, tree, entries, treeObs.Evidence)
		if err != nil {
			return documentContent{}, err
		}
		contentCache[documentPath] = content
		return content, nil
	}

	documentPaths := sortedUnique(append(append([]string{}, profile.Documents...), profile.README.Path))
	for _, documentPath := range documentPaths {
		present := documentationPresence(documentPath, tree, entries, treeObs.Evidence)
		trust := classifyTrustFact(documentPath, router, routerState, routerEvidence)
		inventory.Documents = append(inventory.Documents, DocumentationDocumentFact{Path: documentPath, Present: present, TrustTier: trust})
		if documentPath == profile.README.Path {
			inventory.READMEPresent = present
		}
	}
	if inventory.READMEPresent.State == "" {
		inventory.READMEPresent = documentationPresence(profile.README.Path, tree, entries, treeObs.Evidence)
	}

	readmeContent, err := loadContent(profile.README.Path)
	if err != nil {
		return DocumentationInventory{}, err
	}
	headings := map[string]struct{}{}
	links := map[string]struct{}{}
	if readmeContent.State == StateObserved && readmeContent.Data != nil {
		headings = markdownHeadings(readmeContent.Data)
		links = markdownRepositoryLinks(profile.README.Path, readmeContent.Data)
	}
	for _, section := range profile.README.Sections {
		fact := contentMatchFact(readmeContent, func() bool {
			_, ok := headings[normalizeHeading(section)]
			return ok
		})
		inventory.READMESections = append(inventory.READMESections, DocumentationSectionFact{Section: section, Present: fact})
	}
	for _, target := range profile.README.Links {
		fact := contentMatchFact(readmeContent, func() bool {
			_, ok := links[target]
			return ok
		})
		inventory.READMELinks = append(inventory.READMELinks, DocumentationLinkFact{Target: target, Present: fact})
	}

	for _, marker := range profile.ContentMarkers {
		content, err := loadContent(marker.Path)
		if err != nil {
			return DocumentationInventory{}, err
		}
		digest := sha256.Sum256([]byte(marker.Contains))
		fact := contentMatchFact(content, func() bool { return strings.Contains(string(content.Data), marker.Contains) })
		inventory.ContentMarkers = append(inventory.ContentMarkers, DocumentationMarkerFact{
			ID:             marker.ID,
			Path:           marker.Path,
			ExpectedSHA256: hex.EncodeToString(digest[:]),
			Present:        fact,
		})
	}

	return inventory, inventory.Validate()
}

func loadDocumentationProfile(ctx context.Context, reader GitHubReader, fullName string, tree GitHubTree, entries map[string]GitHubTreeEntry, treeEvidence Evidence) (DocumentationProfile, Evidence, FactState, Fact[bool], error) {
	entry, exists := entries[documentationProfilePath]
	if !exists {
		if tree.Truncated {
			evidence := evidenceWithDetail(treeEvidence, "Git tree is truncated; documentation profile presence is unknown")
			return DocumentationProfile{}, evidence, StateUnknown, Unknown[bool](evidence), nil
		}
		profile := DefaultDocumentationProfile()
		evidence := Evidence{Source: "repora.builtin", Reference: "documentation-profile:baseline"}
		return profile, evidence, StateObserved, Observed(false, treeEvidence), nil
	}
	if entry.Type != "blob" {
		evidence := evidenceWithDetail(treeEvidence, "documentation profile path exists but is not a blob")
		return DocumentationProfile{}, evidence, StateUnknown, Observed(true, treeEvidence), nil
	}
	data, obs, err := reader.Blob(ctx, fullName, entry.SHA)
	if err != nil {
		return DocumentationProfile{}, Evidence{}, "", Fact[bool]{}, err
	}
	if !obs.Available {
		return DocumentationProfile{}, obs.Evidence, StateUnavailable, Observed(true, treeEvidence), nil
	}
	profile, err := ParseDocumentationProfile(data)
	if err != nil {
		evidence := evidenceWithDetail(obs.Evidence, sanitizeDocumentationError(err))
		return DocumentationProfile{}, evidence, StateUnknown, Observed(true, treeEvidence), nil
	}
	return profile, obs.Evidence, StateObserved, Observed(true, treeEvidence), nil
}

func loadTrustRouter(ctx context.Context, reader GitHubReader, fullName string, tree GitHubTree, entries map[string]GitHubTreeEntry, treeEvidence Evidence) (trustRouter, FactState, Evidence, Fact[bool], Fact[bool], error) {
	entry, exists := entries[documentRouterPath]
	if !exists {
		if tree.Truncated {
			evidence := evidenceWithDetail(treeEvidence, "Git tree is truncated; document router presence is unknown")
			return trustRouter{}, StateUnknown, evidence, Unknown[bool](evidence), Unknown[bool](evidence), nil
		}
		evidence := evidenceWithDetail(treeEvidence, "document router is not present")
		return trustRouter{}, StateUnknown, evidence, Observed(false, treeEvidence), Unknown[bool](evidence), nil
	}
	if entry.Type != "blob" {
		evidence := evidenceWithDetail(treeEvidence, "document router path exists but is not a blob")
		return trustRouter{}, StateUnknown, evidence, Observed(true, treeEvidence), Observed(false, evidence), nil
	}
	data, obs, err := reader.Blob(ctx, fullName, entry.SHA)
	if err != nil {
		return trustRouter{}, "", Evidence{}, Fact[bool]{}, Fact[bool]{}, err
	}
	if !obs.Available {
		return trustRouter{}, StateUnavailable, obs.Evidence, Observed(true, treeEvidence), Unavailable[bool](obs.Evidence), nil
	}
	if len(data) > maxDocumentationBytes {
		evidence := evidenceWithDetail(obs.Evidence, fmt.Sprintf("document router exceeds %d-byte normalization limit", maxDocumentationBytes))
		return trustRouter{}, StateUnknown, evidence, Observed(true, treeEvidence), Observed(false, evidence), nil
	}
	router, err := parseTrustRouter(data)
	if err != nil {
		evidence := evidenceWithDetail(obs.Evidence, sanitizeDocumentationError(err))
		return trustRouter{}, StateUnknown, evidence, Observed(true, treeEvidence), Observed(false, evidence), nil
	}
	return router, StateObserved, obs.Evidence, Observed(true, treeEvidence), Observed(true, obs.Evidence), nil
}

func parseTrustRouter(data []byte) (trustRouter, error) {
	var raw struct {
		Version int    `yaml:"version"`
		Kind    string `yaml:"kind"`
		Trust   struct {
			Rules []struct {
				Tier  string   `yaml:"tier"`
				Paths []string `yaml:"paths"`
			} `yaml:"rules"`
		} `yaml:"trust"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return trustRouter{}, fmt.Errorf("parse document router: %w", err)
	}
	if raw.Version != 1 || raw.Kind != "document-router" {
		return trustRouter{}, fmt.Errorf("unsupported document router contract: kind=%q version=%d", raw.Kind, raw.Version)
	}
	validTiers := map[string]bool{"canonical": true, "implementation": true, "generated": true, "experimental": true, "archived": true, "external": true}
	seenPatterns := map[string]struct{}{}
	router := trustRouter{Rules: []trustRule{}}
	for index, rawRule := range raw.Trust.Rules {
		if !validTiers[rawRule.Tier] {
			return trustRouter{}, fmt.Errorf("document router contains unknown trust tier %q", rawRule.Tier)
		}
		if len(rawRule.Paths) == 0 {
			return trustRouter{}, fmt.Errorf("document router trust tier %q has no paths", rawRule.Tier)
		}
		for _, pattern := range rawRule.Paths {
			if pattern == "" {
				return trustRouter{}, fmt.Errorf("document router contains an empty trust pattern")
			}
			if strings.ContainsAny(pattern, "[]") {
				return trustRouter{}, fmt.Errorf("document router trust pattern %q uses unsupported character-class syntax", pattern)
			}
			if _, exists := seenPatterns[pattern]; exists {
				return trustRouter{}, fmt.Errorf("document router contains duplicate trust pattern %q", pattern)
			}
			seenPatterns[pattern] = struct{}{}
		}
		router.Rules = append(router.Rules, trustRule{Tier: rawRule.Tier, Patterns: append([]string(nil), rawRule.Paths...), Index: index})
	}
	if len(router.Rules) == 0 {
		return trustRouter{}, fmt.Errorf("document router contains no trust rules")
	}
	return router, nil
}

func classifyTrustFact(documentPath string, router trustRouter, routerState FactState, evidence Evidence) Fact[string] {
	if routerState == StateUnavailable {
		return Unavailable[string](evidence)
	}
	if routerState != StateObserved {
		return Unknown[string](evidence)
	}
	tier, err := router.classify(documentPath)
	if err != nil {
		return Unknown[string](evidenceWithDetail(evidence, sanitizeDocumentationError(err)))
	}
	return Observed(tier, evidence)
}

func (r trustRouter) classify(candidate string) (string, error) {
	type match struct {
		literal int
		length  int
		index   int
		tier    string
	}
	var best *match
	for _, rule := range r.Rules {
		for _, pattern := range rule.Patterns {
			matched, err := fnmatchCase(pattern, candidate)
			if err != nil {
				return "", err
			}
			if !matched {
				continue
			}
			literal := len(strings.NewReplacer("*", "", "?", "").Replace(pattern))
			current := match{literal: literal, length: len(pattern), index: rule.Index, tier: rule.Tier}
			if best == nil || current.literal > best.literal ||
				(current.literal == best.literal && current.length > best.length) ||
				(current.literal == best.literal && current.length == best.length && current.index < best.index) {
				copy := current
				best = &copy
			}
		}
	}
	if best == nil {
		return "unclassified", nil
	}
	return best.tier, nil
}

func fnmatchCase(pattern, candidate string) (bool, error) {
	var expression strings.Builder
	expression.WriteString("^")
	for _, r := range pattern {
		switch r {
		case '*':
			expression.WriteString(".*")
		case '?':
			expression.WriteString(".")
		case '[', ']':
			return false, fmt.Errorf("trust pattern %q uses unsupported character-class syntax", pattern)
		default:
			expression.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	expression.WriteString("$")
	compiled, err := regexp.Compile(expression.String())
	if err != nil {
		return false, fmt.Errorf("compile trust pattern %q: %w", pattern, err)
	}
	return compiled.MatchString(candidate), nil
}

func documentationPresence(documentPath string, tree GitHubTree, entries map[string]GitHubTreeEntry, evidence Evidence) Fact[bool] {
	entry, exists := entries[documentPath]
	if exists {
		if entry.Type == "blob" {
			return Observed(true, evidence)
		}
		return Observed(false, evidenceWithDetail(evidence, fmt.Sprintf("%s exists but is not a blob", documentPath)))
	}
	if tree.Truncated {
		return Unknown[bool](evidenceWithDetail(evidence, fmt.Sprintf("Git tree is truncated; %s presence is unknown", documentPath)))
	}
	return Observed(false, evidence)
}

func loadDocumentationContent(ctx context.Context, reader GitHubReader, fullName, documentPath string, tree GitHubTree, entries map[string]GitHubTreeEntry, treeEvidence Evidence) (documentContent, error) {
	entry, exists := entries[documentPath]
	if !exists {
		if tree.Truncated {
			return documentContent{State: StateUnknown, Evidence: evidenceWithDetail(treeEvidence, fmt.Sprintf("Git tree is truncated; %s presence is unknown", documentPath))}, nil
		}
		return documentContent{State: StateObserved, Evidence: treeEvidence}, nil
	}
	if entry.Type != "blob" {
		return documentContent{State: StateObserved, Evidence: evidenceWithDetail(treeEvidence, fmt.Sprintf("%s exists but is not a blob", documentPath))}, nil
	}
	data, obs, err := reader.Blob(ctx, fullName, entry.SHA)
	if err != nil {
		return documentContent{}, err
	}
	if !obs.Available {
		return documentContent{State: StateUnavailable, Evidence: obs.Evidence}, nil
	}
	if len(data) > maxDocumentationBytes {
		return documentContent{State: StateUnknown, Evidence: evidenceWithDetail(obs.Evidence, fmt.Sprintf("%s exceeds %d-byte normalization limit", documentPath, maxDocumentationBytes))}, nil
	}
	return documentContent{State: StateObserved, Data: data, Evidence: obs.Evidence}, nil
}

func contentMatchFact(content documentContent, match func() bool) Fact[bool] {
	switch content.State {
	case StateObserved:
		if content.Data == nil {
			return Observed(false, content.Evidence)
		}
		return Observed(match(), content.Evidence)
	case StateUnavailable:
		return Unavailable[bool](content.Evidence)
	default:
		return Unknown[bool](content.Evidence)
	}
}

func markdownHeadings(data []byte) map[string]struct{} {
	result := map[string]struct{}{}
	inFence := false
	fence := ""
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			marker := trimmed[:3]
			if !inFence {
				inFence = true
				fence = marker
			} else if marker == fence {
				inFence = false
				fence = ""
			}
			continue
		}
		if inFence {
			continue
		}
		candidate := strings.TrimLeft(line, " \t")
		if !strings.HasPrefix(candidate, "#") {
			continue
		}
		count := 0
		for count < len(candidate) && candidate[count] == '#' {
			count++
		}
		if count == 0 || count > 6 || count >= len(candidate) || candidate[count] != ' ' {
			continue
		}
		heading := normalizeHeading(strings.TrimSpace(candidate[count+1:]))
		if heading != "" {
			result[heading] = struct{}{}
		}
	}
	return result
}

func normalizeHeading(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimSpace(strings.TrimRight(value, "#"))
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func markdownRepositoryLinks(readmePath string, data []byte) map[string]struct{} {
	result := map[string]struct{}{}
	for _, match := range inlineMarkdownLink.FindAllStringSubmatch(string(data), -1) {
		if len(match) != 2 {
			continue
		}
		target := strings.TrimSpace(match[1])
		if target == "" || strings.HasPrefix(target, "#") || strings.Contains(target, "://") || strings.HasPrefix(strings.ToLower(target), "mailto:") {
			continue
		}
		if index := strings.IndexAny(target, "?#"); index >= 0 {
			target = target[:index]
		}
		if target == "" || strings.HasPrefix(target, "/") {
			continue
		}
		resolved := path.Clean(path.Join(path.Dir(readmePath), target))
		if resolved == "." || resolved == ".." || strings.HasPrefix(resolved, "../") {
			continue
		}
		result[resolved] = struct{}{}
	}
	return result
}

func factForState[T any](state FactState, evidence Evidence) Fact[T] {
	if state == StateUnavailable {
		return Unavailable[T](evidence)
	}
	return Unknown[T](evidence)
}

func evidenceWithDetail(evidence Evidence, detail string) Evidence {
	evidence.Detail = strings.TrimSpace(detail)
	return evidence
}

func sanitizeDocumentationError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.ReplaceAll(err.Error(), "\n", " ")
	message = strings.Join(strings.Fields(message), " ")
	if len(message) > 512 {
		message = message[:512] + "…"
	}
	return message
}

func setDocumentationUnavailable(inventory *DocumentationInventory, evidence Evidence) {
	inventory.DefaultBranch = Unavailable[string](evidence)
	setDocumentationAfterBranchUnavailable(inventory, evidence)
}

func setDocumentationAfterBranchUnavailable(inventory *DocumentationInventory, evidence Evidence) {
	inventory.DefaultCommit = Unavailable[string](evidence)
	setDocumentationAfterTreeUnavailable(inventory, evidence)
}

func setDocumentationAfterTreeUnavailable(inventory *DocumentationInventory, evidence Evidence) {
	inventory.ProfileDeclared = Unavailable[bool](evidence)
	inventory.ProfileName = Unavailable[string](evidence)
	inventory.READMEPresent = Unavailable[bool](evidence)
	inventory.RoutingMetadataPresent = Unavailable[bool](evidence)
	inventory.RoutingMetadataValid = Unavailable[bool](evidence)
}
