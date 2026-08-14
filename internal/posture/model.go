// Package posture collects normalized read-only repository and CI/CD posture facts.
package posture

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	InventoryKind    = "repora.posture-inventory"
	InventoryVersion = 1
)

type FactState string

const (
	StateObserved    FactState = "observed"
	StateUnknown     FactState = "unknown"
	StateUnavailable FactState = "unavailable"
)

type Evidence struct {
	Source    string `json:"source"`
	Reference string `json:"reference"`
	Detail    string `json:"detail,omitempty"`
}

type Fact[T any] struct {
	State    FactState  `json:"state"`
	Value    *T         `json:"value,omitempty"`
	Evidence []Evidence `json:"evidence"`
}

func Observed[T any](value T, evidence ...Evidence) Fact[T] {
	return Fact[T]{State: StateObserved, Value: &value, Evidence: cloneEvidence(evidence)}
}

func Unknown[T any](evidence ...Evidence) Fact[T] {
	return Fact[T]{State: StateUnknown, Evidence: cloneEvidence(evidence)}
}

func Unavailable[T any](evidence ...Evidence) Fact[T] {
	return Fact[T]{State: StateUnavailable, Evidence: cloneEvidence(evidence)}
}

func cloneEvidence(evidence []Evidence) []Evidence {
	if evidence == nil {
		return []Evidence{}
	}
	return append([]Evidence(nil), evidence...)
}

type RepositoryIdentity struct {
	Provider string `json:"provider"`
	FullName string `json:"full_name"`
}

type RepositoryFacts struct {
	DefaultBranch              Fact[string]   `json:"default_branch"`
	DefaultBranchProtected     Fact[bool]     `json:"default_branch_protected"`
	RequiredStatusChecks       Fact[[]string] `json:"required_status_checks"`
	RequiredReviews            Fact[int]      `json:"required_reviews"`
	ForcePushProtected         Fact[bool]     `json:"force_push_protected"`
	DeletionProtected          Fact[bool]     `json:"deletion_protected"`
	CODEOWNERSPresent          Fact[bool]     `json:"codeowners_present"`
	SecurityMDPresent          Fact[bool]     `json:"security_md_present"`
	LicensePresent             Fact[bool]     `json:"license_present"`
	IssueTemplatePresent       Fact[bool]     `json:"issue_template_present"`
	PullRequestTemplatePresent Fact[bool]     `json:"pull_request_template_present"`
	DependencyAutomation       Fact[[]string] `json:"dependency_update_automation"`
	WorkflowPaths              Fact[[]string] `json:"workflow_paths"`
}

type PermissionScope struct {
	Scope  string `json:"scope"`
	Access string `json:"access"`
}

type Permissions struct {
	Declared bool              `json:"declared"`
	Default  string            `json:"default,omitempty"`
	Scopes   []PermissionScope `json:"scopes"`
}

type ActionReference struct {
	Uses       string `json:"uses"`
	ThirdParty bool   `json:"third_party"`
	Pinning    string `json:"pinning"`
}

type WorkflowJob struct {
	Name        string            `json:"name"`
	Permissions Permissions       `json:"permissions"`
	RunsOn      []string          `json:"runs_on"`
	SelfHosted  Fact[bool]        `json:"self_hosted"`
	Actions     []ActionReference `json:"actions"`
}

type Workflow struct {
	Path                  string        `json:"path"`
	State                 FactState     `json:"state"`
	Permissions           Permissions   `json:"permissions"`
	UsesPullRequestTarget bool          `json:"uses_pull_request_target"`
	Jobs                  []WorkflowJob `json:"jobs"`
	Evidence              []Evidence    `json:"evidence"`
}

type Inventory struct {
	Kind            string             `json:"kind"`
	Version         int                `json:"version"`
	Repository      RepositoryIdentity `json:"repository"`
	RepositoryFacts RepositoryFacts    `json:"repository_facts"`
	WorkflowsState  FactState          `json:"workflows_state"`
	Workflows       []Workflow         `json:"workflows"`
	Evidence        []Evidence         `json:"evidence"`
}

func NewInventory(fullName string) Inventory {
	return Inventory{
		Kind:       InventoryKind,
		Version:    InventoryVersion,
		Repository: RepositoryIdentity{Provider: "github", FullName: fullName},
		Workflows:  []Workflow{},
		Evidence:   []Evidence{},
	}
}

func (i Inventory) Validate() error {
	if i.Kind != InventoryKind || i.Version != InventoryVersion {
		return fmt.Errorf("unsupported posture inventory contract: kind=%q version=%d", i.Kind, i.Version)
	}
	if i.Repository.Provider != "github" {
		return fmt.Errorf("posture inventory provider must be github")
	}
	if _, _, err := splitGitHubFullName(i.Repository.FullName); err != nil {
		return err
	}
	checks := []error{
		validateFact("default_branch", i.RepositoryFacts.DefaultBranch),
		validateFact("default_branch_protected", i.RepositoryFacts.DefaultBranchProtected),
		validateFact("required_status_checks", i.RepositoryFacts.RequiredStatusChecks),
		validateFact("required_reviews", i.RepositoryFacts.RequiredReviews),
		validateFact("force_push_protected", i.RepositoryFacts.ForcePushProtected),
		validateFact("deletion_protected", i.RepositoryFacts.DeletionProtected),
		validateFact("codeowners_present", i.RepositoryFacts.CODEOWNERSPresent),
		validateFact("security_md_present", i.RepositoryFacts.SecurityMDPresent),
		validateFact("license_present", i.RepositoryFacts.LicensePresent),
		validateFact("issue_template_present", i.RepositoryFacts.IssueTemplatePresent),
		validateFact("pull_request_template_present", i.RepositoryFacts.PullRequestTemplatePresent),
		validateFact("dependency_update_automation", i.RepositoryFacts.DependencyAutomation),
		validateFact("workflow_paths", i.RepositoryFacts.WorkflowPaths),
	}
	for _, err := range checks {
		if err != nil {
			return err
		}
	}
	if !validState(i.WorkflowsState) {
		return fmt.Errorf("workflows_state %q is invalid", i.WorkflowsState)
	}
	if i.Workflows == nil || i.Evidence == nil {
		return fmt.Errorf("posture inventory workflows and evidence arrays are required")
	}
	for idx, workflow := range i.Workflows {
		if err := workflow.validate(); err != nil {
			return fmt.Errorf("workflow[%d]: %w", idx, err)
		}
	}
	return nil
}

func validateFact[T any](name string, fact Fact[T]) error {
	if !validState(fact.State) {
		return fmt.Errorf("fact %s has invalid state %q", name, fact.State)
	}
	if fact.Evidence == nil {
		return fmt.Errorf("fact %s evidence array is required", name)
	}
	if fact.State == StateObserved && fact.Value == nil {
		return fmt.Errorf("fact %s observed state requires value", name)
	}
	if fact.State != StateObserved && fact.Value != nil {
		return fmt.Errorf("fact %s state %q must not carry a value", name, fact.State)
	}
	return nil
}

func validState(state FactState) bool {
	return state == StateObserved || state == StateUnknown || state == StateUnavailable
}

func (w Workflow) validate() error {
	if strings.TrimSpace(w.Path) == "" {
		return fmt.Errorf("workflow path is required")
	}
	if !validState(w.State) {
		return fmt.Errorf("workflow %q has invalid state %q", w.Path, w.State)
	}
	if w.Jobs == nil || w.Evidence == nil || w.Permissions.Scopes == nil {
		return fmt.Errorf("workflow %q arrays are required", w.Path)
	}
	for idx, job := range w.Jobs {
		if strings.TrimSpace(job.Name) == "" {
			return fmt.Errorf("workflow %q job[%d] name is required", w.Path, idx)
		}
		if job.RunsOn == nil || job.Actions == nil || job.Permissions.Scopes == nil {
			return fmt.Errorf("workflow %q job %q arrays are required", w.Path, job.Name)
		}
		if err := validateFact("workflow job self_hosted", job.SelfHosted); err != nil {
			return err
		}
	}
	return nil
}

func (i Inventory) Marshal() ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode posture inventory: %w", err)
	}
	return append(data, '\n'), nil
}

func sortedUnique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		seen[value] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func splitGitHubFullName(fullName string) (string, string, error) {
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 || !validGitHubNamePart(parts[0]) || !validGitHubNamePart(parts[1]) {
		return "", "", fmt.Errorf("GitHub repository must be OWNER/REPO using letters, digits, '.', '_' or '-': %q", fullName)
	}
	return parts[0], parts[1], nil
}

func validGitHubNamePart(value string) bool {
	if value == "" || value == "." || value == ".." {
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
