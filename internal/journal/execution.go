package journal

import (
	"fmt"
	"reflect"
	"strings"

	"repoctl/internal/executor"
	"repoctl/internal/planartifact"
)

// FromExecution projects one executor result into a validated APPLY journal
// record. It does not persist the record or alter executor behavior.
func FromExecution(executionID string, artifact planartifact.Artifact, result executor.Result) (Record, error) {
	record, err := FromPlan(executionID, ModeApply, artifact)
	if err != nil {
		return Record{}, err
	}
	if len(result.Actions) != len(record.Actions) {
		return Record{}, fmt.Errorf("executor result has %d actions, want %d", len(result.Actions), len(record.Actions))
	}

	plans, err := artifact.Plans()
	if err != nil {
		return Record{}, fmt.Errorf("validate plan artifact: %w", err)
	}
	planned := plans[0]

	for i, executed := range result.Actions {
		if executed.Index != i {
			return Record{}, fmt.Errorf("executor action %d has non-deterministic index %d", i, executed.Index)
		}
		if !reflect.DeepEqual(executed.Action, planned.Actions[i]) {
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

func safeDiagnostic(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "execution failed"
	}
	if unsafeValue(value) {
		return "execution diagnostic redacted"
	}
	return value
}
