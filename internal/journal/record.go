// Package journal defines durable, versioned execution evidence for repository
// reconciliation.
package journal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"repoctl/internal/planartifact"
)

const (
	LegacyVersion = 1
	Version       = 2
	PathVersion   = 3
	Kind          = "repora.io/execution-record"
)

type Phase string

const (
	PhaseIntent Phase = "INTENT"
	PhaseResult Phase = "RESULT"
)

type Mode string

const (
	ModePlan   Mode = "PLAN"
	ModeDryRun Mode = "DRY_RUN"
	ModeApply  Mode = "APPLY"
)

type Outcome string

const (
	OutcomePlanned   Outcome = "PLANNED"
	OutcomeValidated Outcome = "VALIDATED"
	OutcomeApplied   Outcome = "APPLIED"
	OutcomeFailed    Outcome = "FAILED"
	OutcomeSkipped   Outcome = "SKIPPED"
	OutcomeStale     Outcome = "STALE"
)

var (
	identifierPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	oidPattern        = regexp.MustCompile(`^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`)
	digestPattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type Record struct {
	Version     int        `json:"version"`
	Kind        string     `json:"kind"`
	ExecutionID string     `json:"execution_id"`
	Phase       Phase      `json:"phase,omitempty"`
	Mode        Mode       `json:"mode"`
	Plan        PlanRef    `json:"plan"`
	Repository  Repository `json:"repository"`
	Actions     []Action   `json:"actions"`
}

type PlanRef struct {
	Version int    `json:"version"`
	Kind    string `json:"kind"`
	SHA256  string `json:"sha256"`
}

type Repository struct {
	UID string `json:"uid"`
	ID  string `json:"id"`
}

type Ref struct {
	Provider string `json:"provider"`
	Path     string `json:"path,omitempty"`
	Remote   string `json:"remote"`
	Branch   string `json:"branch"`
}

type Action struct {
	Index   int     `json:"index"`
	Type    string  `json:"type"`
	Source  Ref     `json:"source"`
	Target  Ref     `json:"target"`
	Before  string  `json:"before"`
	Desired string  `json:"desired"`
	After   string  `json:"after,omitempty"`
	Force   bool    `json:"force"`
	Outcome Outcome `json:"outcome"`
	Error   string  `json:"error,omitempty"`
}

func FromPlan(executionID string, mode Mode, artifact planartifact.Artifact) (Record, error) {
	encoded, err := artifact.Marshal()
	if err != nil {
		return Record{}, fmt.Errorf("serialize plan artifact: %w", err)
	}
	if len(artifact.Repositories) != 1 {
		return Record{}, fmt.Errorf("journal record requires exactly one repository, got %d", len(artifact.Repositories))
	}

	recordVersion := Version
	if artifact.Version == planartifact.Version {
		recordVersion = PathVersion
	}
	digest := sha256.Sum256(encoded)
	repo := artifact.Repositories[0]
	record := Record{
		Version:     recordVersion,
		Kind:        Kind,
		ExecutionID: executionID,
		Phase:       PhaseIntent,
		Mode:        mode,
		Plan: PlanRef{
			Version: artifact.Version,
			Kind:    artifact.Kind,
			SHA256:  hex.EncodeToString(digest[:]),
		},
		Repository: Repository{UID: repo.UID, ID: repo.ID},
		Actions:    make([]Action, 0, len(repo.Actions)),
	}
	for i, planned := range repo.Actions {
		record.Actions = append(record.Actions, Action{
			Index: i,
			Type:  planned.Type,
			Source: Ref{
				Provider: planned.Source.Provider,
				Path:     planned.Source.Path,
				Remote:   planned.Source.Remote,
				Branch:   planned.Source.Branch,
			},
			Target: Ref{
				Provider: planned.Target.Provider,
				Path:     planned.Target.Path,
				Remote:   planned.Target.Remote,
				Branch:   planned.Target.Branch,
			},
			Before:  planned.Diff.Observed,
			Desired: planned.Diff.Desired,
			Force:   planned.Force,
			Outcome: OutcomePlanned,
		})
	}
	if err := record.Validate(); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (r Record) Marshal() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.MarshalIndent(r, "", "  ")
}

func Parse(data []byte) (Record, error) {
	var record Record
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return Record{}, fmt.Errorf("decode execution record: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Record{}, fmt.Errorf("decode execution record: trailing JSON value")
		}
		return Record{}, fmt.Errorf("decode execution record: trailing data: %w", err)
	}
	if err := record.Validate(); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (r Record) Validate() error {
	if r.Version != LegacyVersion && r.Version != Version && r.Version != PathVersion {
		return fmt.Errorf("unsupported execution record version %d", r.Version)
	}
	if r.Kind != Kind {
		return fmt.Errorf("unsupported execution record kind %q", r.Kind)
	}
	if !validIdentifier(r.ExecutionID) {
		return fmt.Errorf("execution_id must be a symbolic identifier")
	}
	if r.Version == LegacyVersion {
		if r.Phase != "" {
			return fmt.Errorf("legacy execution record must not define a phase")
		}
	} else if !validPhase(r.Phase) {
		return fmt.Errorf("unsupported execution phase %q", r.Phase)
	}
	if !validMode(r.Mode) {
		return fmt.Errorf("unsupported execution mode %q", r.Mode)
	}
	if !planartifact.SupportedVersion(r.Plan.Version) || r.Plan.Kind != planartifact.Kind || !digestPattern.MatchString(r.Plan.SHA256) {
		return fmt.Errorf("plan reference must identify a supported artifact with a SHA-256 digest")
	}
	if !validIdentifier(r.Repository.UID) || !validIdentifier(r.Repository.ID) {
		return fmt.Errorf("repository requires valid uid and id")
	}
	for i, action := range r.Actions {
		if action.Index != i {
			return fmt.Errorf("action %d has non-deterministic index %d", i, action.Index)
		}
		if action.Type != "PUSH_BRANCH" {
			return fmt.Errorf("action %d has unsupported type %q", i, action.Type)
		}
		if err := validateRef(r.Version, action.Source); err != nil {
			return fmt.Errorf("action %d source: %w", i, err)
		}
		if err := validateRef(r.Version, action.Target); err != nil {
			return fmt.Errorf("action %d target: %w", i, err)
		}
		if !oidPattern.MatchString(action.Before) || !oidPattern.MatchString(action.Desired) {
			return fmt.Errorf("action %d requires valid before and desired object IDs", i)
		}
		if action.After != "" && !oidPattern.MatchString(action.After) {
			return fmt.Errorf("action %d has invalid after object ID", i)
		}
		if !validOutcome(r.Version, action.Outcome) {
			return fmt.Errorf("action %d has unsupported outcome %q", i, action.Outcome)
		}
		if action.Outcome == OutcomeApplied && action.After == "" {
			return fmt.Errorf("action %d applied outcome requires after object ID", i)
		}
		if action.Error != "" && unsafeValue(action.Error) {
			return fmt.Errorf("action %d error contains unsafe serialized data", i)
		}
		if r.Version >= Version {
			if err := validatePhaseAction(r.Phase, r.Mode, action); err != nil {
				return fmt.Errorf("action %d: %w", i, err)
			}
		}
	}
	return nil
}

func validatePhaseAction(phase Phase, mode Mode, action Action) error {
	switch phase {
	case PhaseIntent:
		if action.Outcome != OutcomePlanned || action.After != "" || action.Error != "" {
			return fmt.Errorf("intent entry requires planned outcome without after or error")
		}
	case PhaseResult:
		if action.Outcome == OutcomePlanned {
			return fmt.Errorf("result entry must not contain planned outcome")
		}
		if mode == ModeDryRun && action.Outcome == OutcomeApplied {
			return fmt.Errorf("dry-run result must not contain applied outcome")
		}
		if mode == ModeApply && action.Outcome == OutcomeValidated {
			return fmt.Errorf("apply result must not contain validated outcome")
		}
	}
	return nil
}

func validPhase(phase Phase) bool {
	return phase == PhaseIntent || phase == PhaseResult
}

func validMode(mode Mode) bool {
	return mode == ModePlan || mode == ModeDryRun || mode == ModeApply
}

func validOutcome(version int, outcome Outcome) bool {
	switch outcome {
	case OutcomePlanned, OutcomeApplied, OutcomeFailed, OutcomeSkipped, OutcomeStale:
		return true
	case OutcomeValidated:
		return version >= Version
	default:
		return false
	}
}

func validIdentifier(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && identifierPattern.MatchString(value)
}

func validateRef(version int, ref Ref) error {
	if !validIdentifier(ref.Provider) || !validIdentifier(ref.Remote) {
		return fmt.Errorf("provider and remote must be symbolic identifiers")
	}
	if version == PathVersion {
		if err := validateProviderPath(ref.Path); err != nil {
			return err
		}
	} else if strings.TrimSpace(ref.Path) != "" {
		return fmt.Errorf("execution record version %d ref must not define provider path", version)
	}
	return validateBranch(ref.Branch)
}

func validateProviderPath(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || strings.HasPrefix(trimmed, "/") || strings.HasSuffix(trimmed, "/") {
		return fmt.Errorf("provider path must be relative and non-empty")
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 {
		return fmt.Errorf("provider path must include an owner or namespace")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `\:@?#`) || strings.ContainsAny(part, " \t\r\n") {
			return fmt.Errorf("provider path contains an unsafe segment")
		}
	}
	return nil
}

func validateBranch(branch string) error {
	branch = strings.TrimSpace(branch)
	if branch == "" || strings.HasPrefix(branch, "/") || strings.HasSuffix(branch, "/") || strings.HasSuffix(branch, ".") || strings.Contains(branch, "..") || strings.Contains(branch, "//") || strings.Contains(branch, "@{") {
		return fmt.Errorf("branch is not a valid symbolic ref name")
	}
	if strings.ContainsAny(branch, " ~^:?*[\\") {
		return fmt.Errorf("branch is not a valid symbolic ref name")
	}
	for _, segment := range strings.Split(branch, "/") {
		if segment == "" || strings.HasPrefix(segment, ".") || strings.HasSuffix(segment, ".lock") {
			return fmt.Errorf("branch is not a valid symbolic ref name")
		}
	}
	return nil
}

func unsafeValue(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(value, "://") || strings.Contains(value, "token=") || strings.Contains(value, "password=") || strings.Contains(value, "authorization:") || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "file:") || strings.Contains(value, "@")
}
