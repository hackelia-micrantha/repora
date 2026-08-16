package posture

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
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
	maxDocumentationBytes    = 2 << 20
	maxDocumentationTargets  = 256
)

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
	Kind                   string                      `json:"kind"`
	Version                int                         `json:"version"`
	Repository             RepositoryIdentity          `json:"repository"`
	DefaultBranch          Fact[string]                `json:"default_branch"`
	DefaultCommit          Fact[string]                `json:"default_commit"`
	ProfileDeclared        Fact[bool]                  `json:"profile_declared"`
	ProfileName            Fact[string]                `json:"profile_name"`
	READMEPath             string                      `json:"readme_path"`
	READMEPresent          Fact[bool]                  `json:"readme_present"`
	Documents              []DocumentationDocumentFact `json:"documents"`
	READMESections         []DocumentationSectionFact  `json:"readme_sections"`
	READMELinks            []DocumentationLinkFact     `json:"readme_links"`
	ContentMarkers         []DocumentationMarkerFact   `json:"content_markers"`
	RoutingMetadataPresent Fact[bool]                  `json:"routing_metadata_present"`
	RoutingMetadataValid   Fact[bool]                  `json:"routing_metadata_valid"`
	Evidence               []Evidence                  `json:"evidence"`
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
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return DocumentationProfile{}, fmt.Errorf("documentation profile must contain exactly one YAML document")
		}
		return DocumentationProfile{}, fmt.Errorf("parse documentation profile trailing content: %w", err)
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
	checks := []struct {
		name string
		err  error
	}{
		{"default_branch", validateFact("default_branch", i.DefaultBranch)},
		{"default_commit", validateFact("default_commit", i.DefaultCommit)},
		{"profile_declared", validateFact("profile_declared", i.ProfileDeclared)},
		{"profile_name", validateFact("profile_name", i.ProfileName)},
		{"readme_present", validateFact("readme_present", i.READMEPresent)},
		{"routing_metadata_present", validateFact("routing_metadata_present", i.RoutingMetadataPresent)},
		{"routing_metadata_valid", validateFact("routing_metadata_valid", i.RoutingMetadataValid)},
	}
	for _, check := range checks {
		if check.err != nil {
			return fmt.Errorf("%s: %w", check.name, check.err)
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
