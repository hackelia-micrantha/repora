// Package assessment parses and validates versioned repository assessment reports.
package assessment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

const (
	ReportKind    = "repora.repository-assessment"
	ReportVersion = 1
)

var (
	idPattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	commitPattern = regexp.MustCompile(`^[0-9a-fA-F]{7,64}$`)
)

type Report struct {
	Kind      string    `json:"kind"`
	Version   int       `json:"version"`
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Scope     []string  `json:"scope"`
	Summary   string    `json:"summary"`
	Snapshot  Snapshot  `json:"snapshot"`
	Findings  []Finding `json:"findings"`
	Evidence  []Evidence `json:"evidence"`
	Scorecard Scorecard `json:"scorecard"`
	Metadata  *Metadata `json:"metadata,omitempty"`
}

type Snapshot struct {
	Kind       string     `json:"kind"`
	Version    int        `json:"version"`
	Repository Repository `json:"repository"`
	Revision   Revision   `json:"revision"`
	CapturedAt string     `json:"captured_at"`
}

type Repository struct {
	Provider      string `json:"provider,omitempty"`
	FullName      string `json:"full_name"`
	URL           string `json:"url,omitempty"`
	DefaultBranch string `json:"default_branch,omitempty"`
}

type Revision struct {
	Commit string `json:"commit"`
	Dirty  *bool  `json:"dirty"`
}

type Reference struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	Note  string `json:"note,omitempty"`
}

type Evidence struct {
	Kind       string      `json:"kind"`
	Version    int         `json:"version"`
	ID         string      `json:"id"`
	Category   string      `json:"category"`
	Strength   string      `json:"strength"`
	Claim      string      `json:"claim"`
	Rationale  string      `json:"rationale"`
	References []Reference `json:"references"`
}

type Finding struct {
	Kind           string      `json:"kind"`
	Version        int         `json:"version"`
	ID             string      `json:"id"`
	Type           string      `json:"type"`
	Severity       string      `json:"severity"`
	Status         string      `json:"status"`
	Title          string      `json:"title"`
	Description    string      `json:"description"`
	Recommendation string      `json:"recommendation,omitempty"`
	Tradeoffs      []string    `json:"tradeoffs,omitempty"`
	Owner          string      `json:"owner,omitempty"`
	References     []Reference `json:"references"`
	EvidenceIDs    []string    `json:"evidence_ids"`
}

type Scorecard struct {
	Kind       string      `json:"kind"`
	Version    int         `json:"version"`
	Dimensions []Dimension `json:"dimensions"`
}

type Dimension struct {
	Name        string   `json:"name"`
	Score       *int     `json:"score"`
	Rationale   string   `json:"rationale"`
	EvidenceIDs []string `json:"evidence_ids"`
}

type Metadata struct {
	CreatedBy string `json:"created_by,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	Notes     string `json:"notes,omitempty"`
}

var (
	allowedScopes = stringSet("quality", "architecture", "sdlc", "security", "operations", "documentation", "evidence")
	findingTypes = stringSet("question", "finding", "recommendation", "tradeoff", "risk", "gap", "overlap", "drift")
	severities = stringSet("critical", "high", "medium", "low", "informational")
	findingStatuses = stringSet("open", "accepted", "deferred", "implemented", "rejected")
	evidenceCategories = stringSet("architecture", "security", "testing", "devops", "observability", "backend", "mobile", "frontend", "platform", "ai", "leadership", "mentorship")
	evidenceStrengths = stringSet("strong", "moderate", "weak", "unsupported")
	scoreDimensions = stringSet("architecture", "security", "testing", "delivery", "operations", "maintainability", "documentation")
	referenceTypes = stringSet("issue", "pull_request", "commit", "file", "url")
)

func Parse(data []byte) (Report, error) {
	var report Report
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return Report{}, fmt.Errorf("parse assessment: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Report{}, err
	}
	if err := report.Validate(); err != nil {
		return Report{}, err
	}
	return report, nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("parse assessment trailing data: %w", err)
	}
	return fmt.Errorf("parse assessment: multiple JSON values are not allowed")
}

func (r Report) Validate() error {
	if r.Kind != ReportKind || r.Version != ReportVersion {
		return fmt.Errorf("unsupported assessment contract: kind=%q version=%d", r.Kind, r.Version)
	}
	if !validID(r.ID) {
		return fmt.Errorf("assessment id %q is invalid", r.ID)
	}
	if strings.TrimSpace(r.Title) == "" || strings.TrimSpace(r.Summary) == "" {
		return fmt.Errorf("assessment title and summary are required")
	}
	if len(r.Scope) == 0 {
		return fmt.Errorf("assessment scope must be non-empty")
	}
	if err := validateUniqueVocabulary("scope", r.Scope, allowedScopes); err != nil {
		return err
	}
	if err := r.Snapshot.validate(); err != nil {
		return err
	}
	if r.Findings == nil || r.Evidence == nil {
		return fmt.Errorf("assessment findings and evidence arrays are required")
	}

	evidenceIDs := map[string]struct{}{}
	for i, evidence := range r.Evidence {
		if err := evidence.validate(); err != nil {
			return fmt.Errorf("evidence[%d]: %w", i, err)
		}
		if _, exists := evidenceIDs[evidence.ID]; exists {
			return fmt.Errorf("duplicate evidence id %q", evidence.ID)
		}
		evidenceIDs[evidence.ID] = struct{}{}
	}

	findingIDs := map[string]struct{}{}
	for i, finding := range r.Findings {
		if err := finding.validate(evidenceIDs); err != nil {
			return fmt.Errorf("finding[%d]: %w", i, err)
		}
		if _, exists := findingIDs[finding.ID]; exists {
			return fmt.Errorf("duplicate finding id %q", finding.ID)
		}
		findingIDs[finding.ID] = struct{}{}
	}
	if err := r.Scorecard.validate(evidenceIDs); err != nil {
		return err
	}
	if r.Metadata != nil && strings.TrimSpace(r.Metadata.CreatedAt) != "" {
		if _, err := time.Parse(time.RFC3339, r.Metadata.CreatedAt); err != nil {
			return fmt.Errorf("metadata created_at is not RFC3339: %w", err)
		}
	}
	return nil
}

func (s Snapshot) validate() error {
	if s.Kind != "repora.repository-snapshot" || s.Version != 1 {
		return fmt.Errorf("unsupported repository snapshot contract: kind=%q version=%d", s.Kind, s.Version)
	}
	if strings.TrimSpace(s.Repository.FullName) == "" {
		return fmt.Errorf("repository full_name is required")
	}
	if !commitPattern.MatchString(s.Revision.Commit) {
		return fmt.Errorf("snapshot commit must be a 7-64 character hex revision")
	}
	if s.Revision.Dirty == nil {
		return fmt.Errorf("snapshot dirty field is required")
	}
	if _, err := time.Parse(time.RFC3339, s.CapturedAt); err != nil {
		return fmt.Errorf("snapshot captured_at is not RFC3339: %w", err)
	}
	return nil
}

func (e Evidence) validate() error {
	if e.Kind != "repora.evidence" || e.Version != 1 {
		return fmt.Errorf("unsupported evidence contract")
	}
	if !validID(e.ID) {
		return fmt.Errorf("evidence id %q is invalid", e.ID)
	}
	if _, ok := evidenceCategories[e.Category]; !ok {
		return fmt.Errorf("evidence %q has unsupported category %q", e.ID, e.Category)
	}
	if _, ok := evidenceStrengths[e.Strength]; !ok {
		return fmt.Errorf("evidence %q has unsupported strength %q", e.ID, e.Strength)
	}
	if strings.TrimSpace(e.Claim) == "" || strings.TrimSpace(e.Rationale) == "" {
		return fmt.Errorf("evidence %q claim and rationale are required", e.ID)
	}
	if e.References == nil {
		return fmt.Errorf("evidence %q references array is required", e.ID)
	}
	if e.Strength != "unsupported" && len(e.References) == 0 {
		return fmt.Errorf("evidence %q with strength %q requires a reference", e.ID, e.Strength)
	}
	return validateReferences(e.References)
}

func (f Finding) validate(evidenceIDs map[string]struct{}) error {
	if f.Kind != "repora.finding" || f.Version != 1 {
		return fmt.Errorf("unsupported finding contract")
	}
	if !validID(f.ID) {
		return fmt.Errorf("finding id %q is invalid", f.ID)
	}
	if _, ok := findingTypes[f.Type]; !ok {
		return fmt.Errorf("finding %q has unsupported type %q", f.ID, f.Type)
	}
	if _, ok := severities[f.Severity]; !ok {
		return fmt.Errorf("finding %q has unsupported severity %q", f.ID, f.Severity)
	}
	if _, ok := findingStatuses[f.Status]; !ok {
		return fmt.Errorf("finding %q has unsupported status %q", f.ID, f.Status)
	}
	if strings.TrimSpace(f.Title) == "" || strings.TrimSpace(f.Description) == "" {
		return fmt.Errorf("finding %q title and description are required", f.ID)
	}
	if f.References == nil || f.EvidenceIDs == nil {
		return fmt.Errorf("finding %q references and evidence_ids arrays are required", f.ID)
	}
	if err := validateReferences(f.References); err != nil {
		return fmt.Errorf("finding %q: %w", f.ID, err)
	}
	return validateEvidenceLinks("finding "+f.ID, f.EvidenceIDs, evidenceIDs)
}

func (s Scorecard) validate(evidenceIDs map[string]struct{}) error {
	if s.Kind != "repora.scorecard" || s.Version != 1 {
		return fmt.Errorf("unsupported scorecard contract")
	}
	if len(s.Dimensions) == 0 {
		return fmt.Errorf("scorecard dimensions must be non-empty")
	}
	seen := map[string]struct{}{}
	for i, dimension := range s.Dimensions {
		if _, ok := scoreDimensions[dimension.Name]; !ok {
			return fmt.Errorf("scorecard dimension[%d] has unsupported name %q", i, dimension.Name)
		}
		if _, exists := seen[dimension.Name]; exists {
			return fmt.Errorf("duplicate scorecard dimension %q", dimension.Name)
		}
		seen[dimension.Name] = struct{}{}
		if dimension.Score == nil || *dimension.Score < 0 || *dimension.Score > 5 {
			return fmt.Errorf("scorecard %q score must be an integer from 0 through 5", dimension.Name)
		}
		if strings.TrimSpace(dimension.Rationale) == "" {
			return fmt.Errorf("scorecard %q rationale is required", dimension.Name)
		}
		if dimension.EvidenceIDs == nil {
			return fmt.Errorf("scorecard %q evidence_ids array is required", dimension.Name)
		}
		if err := validateEvidenceLinks("scorecard "+dimension.Name, dimension.EvidenceIDs, evidenceIDs); err != nil {
			return err
		}
	}
	return nil
}

func validateReferences(refs []Reference) error {
	for i, ref := range refs {
		if _, ok := referenceTypes[ref.Type]; !ok {
			return fmt.Errorf("reference[%d] has unsupported type %q", i, ref.Type)
		}
		if strings.TrimSpace(ref.Value) == "" {
			return fmt.Errorf("reference[%d] value is required", i)
		}
	}
	return nil
}

func validateEvidenceLinks(context string, ids []string, evidenceIDs map[string]struct{}) error {
	seen := map[string]struct{}{}
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			return fmt.Errorf("%s has duplicate evidence id %q", context, id)
		}
		seen[id] = struct{}{}
		if _, exists := evidenceIDs[id]; !exists {
			return fmt.Errorf("%s references unknown evidence id %q", context, id)
		}
	}
	return nil
}

func validateUniqueVocabulary(name string, values []string, allowed map[string]struct{}) error {
	seen := map[string]struct{}{}
	for _, value := range values {
		if _, ok := allowed[value]; !ok {
			return fmt.Errorf("%s contains unsupported value %q", name, value)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("%s contains duplicate value %q", name, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validID(value string) bool {
	return idPattern.MatchString(value)
}

func stringSet(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}
