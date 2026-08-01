// Package journal defines durable, versioned execution evidence for repository
// reconciliation. Filesystem persistence is intentionally owned by a later slice.
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
	Version = 1
	Kind    = "repora.io/execution-record"
)

type Mode string

const (
	ModePlan   Mode = "PLAN"
	ModeDryRun Mode = "DRY_RUN"
	ModeApply  Mode = "APPLY"
)

type Outcome string

const (
	OutcomePlanned Outcome = "PLANNED"
	OutcomeApplied Outcome = "APPLIED"
	OutcomeFailed  Outcome = "FAILED"
	OutcomeSkipped Outcome = "SKIPPED"
	OutcomeStale   Outcome = "STALE"
)

var (
	identifierPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	oidPattern        = regexp.MustCompile(`^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`)
	digestPattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// Record is the durable evidence envelope for one repository execution.
type Record struct {
	Version     int         `json:"version"`
	Kind        string      `json:"kind"`
	ExecutionID string      `json:"execution_id"`
	Mode        Mode        `json:"mode"`
	Plan        PlanRef     `json:"plan"`
	Repository  Repository  `json:"repository"`
	Actions     []Action    `json:"actions"`
}

// PlanRef identifies the exact serialized plan artifact used for the run.
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
	Remote   string `json:"remote"`
	Branch   string `json:"branch"`
}

// Action records planned ref state and the eventual execution outcome. After is
// optional until a mutation has produced a verified resulting object ID.
type Action struct {
	Index    int     `json:"index"`
	Type     string  `json:"type"`
	Source   Ref     `json:"source"`
	Target   Ref     `json:"target"`
	Before   string  `json:"before"`
	Desired  string  `json:"desired"`
	After    string  `json:"after,omitempty"`
	Force    bool    `json:"force"`
	Outcome  Outcome `json:"outcome"`
	Error    string  `json:"error,omitempty"`
}

// FromPlan creates deterministic planned evidence for exactly one repository.
func FromPlan(executionID string, mode Mode, artifact planartifact.Artifact) (Record, error) {
	encoded, err := artifact.Marshal()
	if err != nil {
		return Record{}, fmt.Errorf("serialize plan artifact: %w", err)
	}
	if len(artifact.Repositories) != 1 {
		return Record{}, fmt.Errorf("journal record requires exactly one repository, got %d", len(artifact.Repositories))
	}

	digest := sha256.Sum256(encoded)
	repo := artifact.Repositories[0]
	record := Record{
		Version:     Version,
		Kind:        Kind,
		ExecutionID: executionID,
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
			Index:   i,
			Type:    planned.Type,
			Source:  Ref{Provider: planned.Source.Provider, Remote: planned.Source.Remote, Branch: planned.Source.Branch},
			Target:  Ref{Provider: planned.Target.Provider, Remote: planned.Target.Remote, Branch: planned.Target.Branch},
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
	if r.Version != Version {
		return fmt.Errorf("unsupported execution record version %d", r.Version)
	}
	if r.Kind != Kind {
		return fmt.Errorf("unsupported execution record kind %q", r.Kind)
	}
	if !validIdentifier(r.ExecutionID) {
		return fmt.Errorf("execution_id must be a symbolic identifier")
	}
	if !validMode(r.Mode) {
		return fmt.Errorf("unsupported execution mode %q", r.Mode)
	}
	if r.Plan.Version != planartifact.Version || r.Plan.Kind != planartifact.Kind || !digestPattern.MatchString(r.Plan.SHA256) {
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
		if err := validateRef(action.Source); err != nil {
			return fmt.Errorf("action %d source: %w", i, err)
		}
		if err := validateRef(action.Target); err != nil {
			return fmt.Errorf("action %d target: %w", i, err)
		}
		if !oidPattern.MatchString(action.Before) || !oidPattern.MatchString(action.Desired) {
			return fmt.Errorf("action %d requires valid before and desired object IDs", i)
		}
		if action.After != "" && !oidPattern.MatchString(action.After) {
			return fmt.Errorf("action %d has invalid after object ID", i)
		}
		if !validOutcome(action.Outcome) {
			return fmt.Errorf("action %d has unsupported outcome %q", i, action.Outcome)
		}
		if action.Outcome == OutcomeApplied && action.After == "" {
			return fmt.Errorf("action %d applied outcome requires after object ID", i)
		}
		if action.Error != "" && unsafeValue(action.Error) {
			return fmt.Errorf("action %d error contains unsafe serialized data", i)
		}
	}
	return nil
}

func validMode(mode Mode) bool {
	return mode == ModePlan || mode == ModeDryRun || mode == ModeApply
}

func validOutcome(outcome Outcome) bool {
	switch outcome {
	case OutcomePlanned, OutcomeApplied, OutcomeFailed, OutcomeSkipped, OutcomeStale:
		return true
	default:
		return false
	}
}

func validIdentifier(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && identifierPattern.MatchString(value)
}

func validateRef(ref Ref) error {
	if !validIdentifier(ref.Provider) || !validIdentifier(ref.Remote) {
		return fmt.Errorf("provider and remote must be symbolic identifiers")
	}
	if strings.TrimSpace(ref.Branch) == "" || unsafeValue(ref.Branch) {
		return fmt.Errorf("branch must be a safe symbolic ref")
	}
	return nil
}

func unsafeValue(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(value, "://") || strings.Contains(value, "token=") || strings.Contains(value, "password=") || strings.Contains(value, "authorization:") || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "file:") || strings.Contains(value, "@")
}
