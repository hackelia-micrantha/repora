package managedartifact

import (
	"bytes"
	"errors"
	"fmt"

	"repoctl/internal/config"
)

// ErrStale identifies a valid managed-artifact plan whose reviewed repository
// state no longer matches the current canonical state or current configuration.
var ErrStale = errors.New("managed artifact plan is stale")

// PreflightPlan rebinds an exact managed-artifact plan to current configuration
// and canonical README state. It performs no repository mutation beyond the
// observer's local cache refresh/fetch behavior.
func PreflightPlan(spec config.Spec, plan Plan, observer READMEObserver) error {
	if err := plan.Validate(); err != nil {
		return fmt.Errorf("validate managed artifact plan: %w", err)
	}
	if len(plan.Repositories) == 0 {
		return nil
	}
	if observer == nil {
		return fmt.Errorf("README observer is required for managed artifact preflight")
	}

	configured := make(map[string]config.Repo, len(spec.Repos))
	for i, repo := range spec.Repos {
		uid := repo.DurableID()
		if _, exists := configured[uid]; exists {
			return fmt.Errorf("configuration contains duplicate repository uid %q at index %d", uid, i)
		}
		configured[uid] = repo
	}

	selected := make([]config.Repo, len(plan.Repositories))
	for i, planned := range plan.Repositories {
		repo, ok := configured[planned.UID]
		if !ok {
			return stalef("repository uid %q is no longer configured", planned.UID)
		}
		if repo.Artifacts.Readme == nil {
			return stalef("repository uid %q no longer enables managed README authority", planned.UID)
		}
		if err := validatePlannerRepositories([]config.Repo{repo}); err != nil {
			return stalef("repository uid %q current configuration is no longer plan-compatible: %v", planned.UID, err)
		}
		if repo.ID != planned.ID {
			return stalef("repository uid %q id changed from %q to %q", planned.UID, planned.ID, repo.ID)
		}
		if repo.Canonical.Provider != planned.Target.Provider || repo.Canonical.Path != planned.Target.Path {
			return stalef("repository uid %q canonical target changed from %s/%s to %s/%s", planned.UID, planned.Target.Provider, planned.Target.Path, repo.Canonical.Provider, repo.Canonical.Path)
		}
		selected[i] = repo
	}

	for i, planned := range plan.Repositories {
		current, err := observer.ObserveREADME(selected[i])
		if err != nil {
			return fmt.Errorf("repo %q: observe current README for preflight: %w", planned.ID, err)
		}
		if err := validateObservation(current); err != nil {
			return fmt.Errorf("repo %q: validate current README observation: %w", planned.ID, err)
		}
		if current.Branch != planned.Target.Branch {
			return stalef("repo %q canonical default branch changed from %q to %q", planned.ID, planned.Target.Branch, current.Branch)
		}
		if current.BaseOID != planned.BaseOID {
			return stalef("repo %q canonical HEAD changed from %s to %s", planned.ID, planned.BaseOID, current.BaseOID)
		}

		action := planned.Actions[0]
		if err := compareObservedState(planned.ID, action.Observed, current); err != nil {
			return err
		}
		desired := []byte(*action.Desired.Content)
		diff, err := ReviewDiff(current.Present, current.Content, desired)
		if err != nil {
			return fmt.Errorf("repo %q: recompute reviewed README diff: %w", planned.ID, err)
		}
		if diff != action.Diff {
			return stalef("repo %q reviewed README diff no longer matches exact current and desired content", planned.ID)
		}
	}
	return nil
}

func compareObservedState(repoID string, expected ObservedState, current READMEObservation) error {
	if expected.Present == nil {
		return fmt.Errorf("repo %q: managed plan observed presence is missing", repoID)
	}
	if *expected.Present != current.Present {
		return stalef("repo %q README presence changed", repoID)
	}
	if !current.Present {
		return nil
	}
	if expected.Mode != current.Mode {
		return stalef("repo %q README mode changed from %s to %s", repoID, expected.Mode, current.Mode)
	}
	currentDigest := DigestSHA256(current.Content)
	if expected.SHA256 != currentDigest {
		return stalef("repo %q README content digest changed from %s to %s", repoID, expected.SHA256, currentDigest)
	}
	return nil
}

func stalef(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrStale, fmt.Sprintf(format, args...))
}

// bytesEqual is kept package-local for tests that need to assert exact raw
// content identity without weakening the serialized plan contract.
func bytesEqual(left, right []byte) bool {
	return bytes.Equal(left, right)
}
