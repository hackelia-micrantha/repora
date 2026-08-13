package managedartifactapply

import (
	"errors"
	"fmt"

	"repoctl/internal/config"
	"repoctl/internal/journal"
	"repoctl/internal/managedartifact"
)

const (
	ResultVersion = 1
	ResultKind    = "repora.io/managed-artifact-apply-result"
)

type Preparer interface {
	Prepare(spec config.Spec, plan managedartifact.Plan, observer managedartifact.READMEObserver) ([]managedartifact.PreparedCommit, error)
}

type Pusher interface {
	Push(spec config.Spec, plan managedartifact.Plan, prepared []managedartifact.PreparedCommit, observer managedartifact.READMEObserver) ([]managedartifact.PushResult, error)
}

type JournalWriter interface {
	WriteManagedArtifact(record journal.ManagedArtifactRecord) (string, error)
}

type Audit struct {
	ExecutionID string
	Writer      JournalWriter
}

type JournalReferences struct {
	Intent string `json:"intent"`
	Result string `json:"result,omitempty"`
}

type RepositoryResult struct {
	UID       string          `json:"uid"`
	ID        string          `json:"id"`
	Branch    string          `json:"branch"`
	BaseOID   string          `json:"base_oid"`
	CommitOID string          `json:"commit_oid,omitempty"`
	Pushed    bool            `json:"pushed"`
	Outcome   journal.Outcome `json:"outcome"`
}

type Result struct {
	Version      int                `json:"version"`
	Kind         string             `json:"kind"`
	ExecutionID  string             `json:"execution_id"`
	Outcome      journal.Outcome    `json:"outcome"`
	FailureStage string             `json:"failure_stage,omitempty"`
	Repositories []RepositoryResult `json:"repositories"`
	Journal      JournalReferences  `json:"journal"`
}

// Reportable is true only after a managed-artifact RESULT has been projected.
// Early failures such as INTENT persistence failure must not be serialized as
// the stabilized apply-result contract because no valid RESULT exists yet.
func (r Result) Reportable() bool {
	if r.Version != ResultVersion || r.Kind != ResultKind || r.ExecutionID == "" || r.Journal.Intent == "" || len(r.Repositories) == 0 {
		return false
	}
	switch r.Outcome {
	case journal.OutcomeApplied, journal.OutcomeFailed, journal.OutcomeStale:
		return true
	default:
		return false
	}
}

// Execute persists INTENT before candidate-object creation, prepares verified
// local commits, performs guarded pushes, then persists RESULT even for stale,
// partial-success, or operational failure outcomes.
func Execute(spec config.Spec, plan managedartifact.Plan, observer managedartifact.READMEObserver, preparer Preparer, pusher Pusher, audit Audit) (Result, error) {
	result := Result{
		Version:      ResultVersion,
		Kind:         ResultKind,
		ExecutionID:  audit.ExecutionID,
		Repositories: []RepositoryResult{},
	}
	if err := plan.Validate(); err != nil {
		return result, fmt.Errorf("validate managed README plan: %w", err)
	}
	if len(plan.Repositories) == 0 {
		result.Outcome = journal.OutcomeSkipped
		return result, nil
	}
	if audit.ExecutionID == "" || audit.Writer == nil {
		return result, fmt.Errorf("managed README apply requires execution ID and journal writer")
	}
	if observer == nil || preparer == nil || pusher == nil {
		return result, fmt.Errorf("managed README apply dependencies are incomplete")
	}

	intent, err := journal.ManagedArtifactIntent(audit.ExecutionID, plan)
	if err != nil {
		return result, fmt.Errorf("create managed README journal intent: %w", err)
	}
	intentRef, err := audit.Writer.WriteManagedArtifact(intent)
	result.Journal.Intent = intentRef
	if err != nil {
		return result, fmt.Errorf("persist managed README journal intent: %w", err)
	}

	prepared, prepareErr := preparer.Prepare(spec, plan, observer)
	if prepareErr != nil {
		return finish(result, plan, prepared, nil, prepareErr, audit.Writer)
	}
	pushes, pushErr := pusher.Push(spec, plan, prepared, observer)
	return finish(result, plan, prepared, pushes, pushErr, audit.Writer)
}

func finish(result Result, plan managedartifact.Plan, prepared []managedartifact.PreparedCommit, pushes []managedartifact.PushResult, executionErr error, writer JournalWriter) (Result, error) {
	record, err := journal.ManagedArtifactResult(result.ExecutionID, plan, prepared, pushes, executionErr)
	if err != nil {
		return result, errors.Join(executionErr, fmt.Errorf("create managed README journal result: %w", err))
	}
	projectRecord(&result, record)
	resultRef, writeErr := writer.WriteManagedArtifact(record)
	result.Journal.Result = resultRef
	if writeErr != nil {
		return result, errors.Join(executionErr, fmt.Errorf("persist managed README journal result: %w", writeErr))
	}
	return result, executionErr
}

func projectRecord(result *Result, record journal.ManagedArtifactRecord) {
	result.Outcome = record.Outcome
	result.FailureStage = record.FailureStage
	result.Repositories = make([]RepositoryResult, 0, len(record.Repositories))
	for _, repo := range record.Repositories {
		result.Repositories = append(result.Repositories, RepositoryResult{
			UID:       repo.UID,
			ID:        repo.ID,
			Branch:    repo.Branch,
			BaseOID:   repo.BaseOID,
			CommitOID: repo.PreparedCommit,
			Pushed:    repo.Pushed,
			Outcome:   repo.Outcome,
		})
	}
}
