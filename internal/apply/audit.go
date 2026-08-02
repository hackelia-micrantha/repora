package apply

import (
	"errors"
	"fmt"
	"strings"

	"repoctl/internal/config"
	"repoctl/internal/executor"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/journal"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

type JournalWriter interface {
	Write(record journal.Record) (string, error)
}

type Audit struct {
	ExecutionID string
	Writer      JournalWriter
}

// ExecuteAudited builds the exact artifact and writes immutable INTENT and
// RESULT entries around dry-run preflight or mutation.
func ExecuteAudited(repo config.Repo, st status.Result, git Git, force bool, dryRun bool, audit Audit) (Result, error) {
	artifact, err := BuildArtifact(repo, st, git)
	if err != nil {
		return newResult(repo, st, dryRun), err
	}
	return ExecuteArtifactAudited(repo, st, artifact, git, force, dryRun, audit)
}

// ExecuteArtifactAudited consumes an exact artifact and guarantees that a
// durable INTENT entry exists before executor preflight can reach mutation.
// It then attempts a RESULT entry for success, stale input, or runtime failure.
func ExecuteArtifactAudited(repo config.Repo, st status.Result, artifact planartifact.Artifact, git Git, allowForce bool, dryRun bool, audit Audit) (Result, error) {
	if strings.TrimSpace(audit.ExecutionID) == "" {
		return newResult(repo, st, dryRun), fmt.Errorf("audit execution ID is required")
	}
	if audit.Writer == nil {
		return newResult(repo, st, dryRun), fmt.Errorf("audit journal writer is required")
	}
	return executeArtifact(repo, st, artifact, git, allowForce, dryRun, &audit)
}

func executeArtifact(repo config.Repo, st status.Result, artifact planartifact.Artifact, git Git, allowForce bool, dryRun bool, audit *Audit) (Result, error) {
	result := newResult(repo, st, dryRun)
	planned, err := planForRepository(repo, artifact)
	if err != nil {
		return result, err
	}
	result.Actions = compatibilityActions(planned)
	if err := validateStateIntent(repo.ID, st.State, planned); err != nil {
		return result, err
	}
	if !dryRun && planRequiresForce(planned) && !allowForce {
		return result, fmt.Errorf("repo %q plan contains a forced action; rerun with --force", repo.ID)
	}

	path := ""
	if len(planned.Actions) > 0 {
		path, err = gitwrap.MirrorPath(repo.DurableID())
		if err != nil {
			return result, err
		}
		if err := validateDefaultBranchScope(path, planned, git); err != nil {
			return result, fmt.Errorf("validate plan scope for repo %q: %w", repo.ID, err)
		}
	}

	mode := journal.ModeApply
	if dryRun {
		mode = journal.ModeDryRun
	}
	if audit != nil {
		intent, err := journal.FromPlan(audit.ExecutionID, mode, artifact)
		if err != nil {
			return result, fmt.Errorf("create journal intent for repo %q: %w", repo.ID, err)
		}
		intentRef, err := audit.Writer.Write(intent)
		result.Journal = &JournalReferences{ExecutionID: audit.ExecutionID, Intent: intentRef}
		if err != nil {
			return result, fmt.Errorf("persist journal intent for repo %q: %w", repo.ID, err)
		}
	}

	if dryRun {
		preflight := executor.Result{Actions: []executor.ActionResult{}}
		var preflightErr error
		if len(planned.Actions) > 0 {
			preflight, preflightErr = executor.Preflight(path, artifact, git)
		}
		journalErr := persistPreflightResult(&result, audit, artifact, preflight, preflightErr)
		if err := errors.Join(preflightErr, journalErr); err != nil {
			return result, fmt.Errorf("dry-run plan artifact for repo %q: %w", repo.ID, err)
		}
		return result, nil
	}

	executed := executor.Result{Actions: []executor.ActionResult{}}
	var executeErr error
	if len(planned.Actions) > 0 {
		executed, executeErr = executor.Execute(path, artifact, git)
	}
	result.Applied = executed.AllApplied()
	journalErr := persistExecutionResult(&result, audit, artifact, executed)
	if err := errors.Join(executeErr, journalErr); err != nil {
		return result, fmt.Errorf("execute plan artifact for repo %q: %w", repo.ID, err)
	}
	return result, nil
}

func persistPreflightResult(result *Result, audit *Audit, artifact planartifact.Artifact, preflight executor.Result, preflightErr error) error {
	if audit == nil {
		return nil
	}
	record, err := journal.FromPreflight(audit.ExecutionID, artifact, preflight, preflightErr)
	if err != nil {
		return fmt.Errorf("create journal result: %w", err)
	}
	ref, err := audit.Writer.Write(record)
	result.Journal.Result = ref
	if err != nil {
		return fmt.Errorf("persist journal result: %w", err)
	}
	return nil
}

func persistExecutionResult(result *Result, audit *Audit, artifact planartifact.Artifact, executed executor.Result) error {
	if audit == nil {
		return nil
	}
	record, err := journal.FromExecution(audit.ExecutionID, artifact, executed)
	if err != nil {
		return fmt.Errorf("create journal result: %w", err)
	}
	ref, err := audit.Writer.Write(record)
	result.Journal.Result = ref
	if err != nil {
		return fmt.Errorf("persist journal result: %w", err)
	}
	return nil
}
