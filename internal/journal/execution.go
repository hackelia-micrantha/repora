package journal

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"repoctl/internal/executor"
	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
)

var embeddedAbsolutePathPattern = regexp.MustCompile(`(^|[[:space:]'"])/[^[:space:]'"]+`)

// FromPreflight projects one dry-run preflight result into a validated RESULT
// entry. Successful actions become VALIDATED; stale or failed actions preserve
// executor detail and later actions remain SKIPPED.
func FromPreflight(executionID string, artifact planartifact.Artifact, result executor.Result, preflightErr error) (Record, error) {
	record, planned, err := resultRecord(executionID, ModeDryRun, artifact, result)
	if err != nil {
		return Record{}, err
	}

	failedAction := false
	for i, executed := range result.Actions {
		if executed.Index != i {
			return Record{}, fmt.Errorf("preflight action %d has non-deterministic index %d", i, executed.Index)
		}
		if !reflect.DeepEqual(executed.Action, planned[i]) {
			return Record{}, fmt.Errorf("preflight action %d does not match the referenced plan", i)
		}

		action := &record.Actions[i]
		if preflightErr == nil {
			if executed.Outcome != executor.OutcomeSkipped || executed.Stale || strings.TrimSpace(executed.Error) != "" {
				return Record{}, fmt.Errorf("preflight action %d contains inconsistent executor evidence for a successful preflight", i)
			}
			action.Outcome = OutcomeValidated
			continue
		}
		switch executed.Outcome {
		case executor.OutcomeFailed:
			failedAction = true
			if executed.Stale {
				action.Outcome = OutcomeStale
			} else {
				action.Outcome = OutcomeFailed
			}
			action.Error = safeDiagnostic(executed.Error)
		case executor.OutcomeSkipped:
			action.Outcome = OutcomeSkipped
		case executor.OutcomeApplied:
			return Record{}, fmt.Errorf("preflight action %d unexpectedly reports applied outcome", i)
		default:
			return Record{}, fmt.Errorf("preflight action %d has unsupported outcome %q", i, executed.Outcome)
		}
	}
	if preflightErr != nil && !failedAction {
		return Record{}, fmt.Errorf("preflight failure does not identify a failed action")
	}

	if err := record.Validate(); err != nil {
		return Record{}, fmt.Errorf("validate preflight record: %w", err)
	}
	return record, nil
}

// FromExecution projects one executor result into a validated APPLY RESULT
// entry. It does not persist the record or alter executor behavior.
func FromExecution(executionID string, artifact planartifact.Artifact, result executor.Result) (Record, error) {
	record, planned, err := resultRecord(executionID, ModeApply, artifact, result)
	if err != nil {
		return Record{}, err
	}

	for i, executed := range result.Actions {
		if executed.Index != i {
			return Record{}, fmt.Errorf("executor action %d has non-deterministic index %d", i, executed.Index)
		}
		if !reflect.DeepEqual(executed.Action, planned[i]) {
			return Record{}, fmt.Errorf("executor action %d does not match the referenced plan", i)
		}

		action := &record.Actions[i]
		switch executed.Outcome {
		case executor.OutcomeApplied:
			action.Outcome = OutcomeApplied
			action.After = strings.TrimSpace(executed.AfterOID)
		case executor.OutcomeFailed:
			if executed.Stale {
				action.Outcome = OutcomeStale
			} else {
				action.Outcome = OutcomeFailed
			}
			action.Error = safeDiagnostic(executed.Error)
		case executor.OutcomeSkipped:
			action.Outcome = OutcomeSkipped
		default:
			return Record{}, fmt.Errorf("executor action %d has unsupported outcome %q", i, executed.Outcome)
		}
	}

	if err := record.Validate(); err != nil {
		return Record{}, fmt.Errorf("validate execution record: %w", err)
	}
	return record, nil
}

func resultRecord(executionID string, mode Mode, artifact planartifact.Artifact, result executor.Result) (Record, []plan.PlannedAction, error) {
	record, err := FromPlan(executionID, mode, artifact)
	if err != nil {
		return Record{}, nil, err
	}
	record.Phase = PhaseResult
	if len(result.Actions) != len(record.Actions) {
		return Record{}, nil, fmt.Errorf("executor result has %d actions, want %d", len(result.Actions), len(record.Actions))
	}

	plans, err := artifact.Plans()
	if err != nil {
		return Record{}, nil, fmt.Errorf("validate plan artifact: %w", err)
	}
	return record, plans[0].Actions, nil
}

func safeDiagnostic(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "execution failed"
	}
	if unsafeValue(value) || embeddedAbsolutePathPattern.MatchString(value) {
		return "execution diagnostic redacted"
	}
	return value
}
