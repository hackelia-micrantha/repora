package managedartifact

import (
	"bytes"
	"fmt"
	"sort"

	"repoctl/internal/config"
)

// READMEObservation is exact canonical README state supplied by a read-only
// repository observer. Content is raw blob content so byte-only changes remain
// reviewable and digest-bound.
type READMEObservation struct {
	Branch  string
	BaseOID string
	Present bool
	Mode    string
	Content []byte
}

// READMEObserver supplies exact canonical README state. Implementations must be
// read-only; Git transport/cache binding is intentionally outside this planner.
type READMEObserver interface {
	ObserveREADME(repo config.Repo) (READMEObservation, error)
}

// BuildPlan constructs a deterministic managed-artifact plan for configured
// README artifacts. Unconfigured and already-equal repositories are omitted;
// an explicit empty repositories array represents no changes.
func BuildPlan(configPath string, spec config.Spec, observer READMEObserver) (Plan, error) {
	plan := Plan{Kind: PlanKind, Version: PlanVersion, Repositories: []RepositoryPlan{}}

	repos := make([]config.Repo, 0, len(spec.Repos))
	for _, repo := range spec.Repos {
		if repo.Artifacts.Readme != nil {
			repos = append(repos, repo)
		}
	}
	if len(repos) == 0 {
		if err := plan.Validate(); err != nil {
			return Plan{}, fmt.Errorf("build managed artifact plan: %w", err)
		}
		return plan, nil
	}
	if err := validatePlannerRepositories(repos); err != nil {
		return Plan{}, err
	}
	if observer == nil {
		return Plan{}, fmt.Errorf("README observer is required for configured managed artifacts")
	}

	sort.Slice(repos, func(i, j int) bool {
		left, right := repos[i].DurableID(), repos[j].DurableID()
		if left != right {
			return left < right
		}
		return repos[i].ID < repos[j].ID
	})

	for _, repo := range repos {
		readmeConfig := repo.Artifacts.Readme
		template, err := LoadTemplate(configPath, readmeConfig.Template)
		if err != nil {
			return Plan{}, fmt.Errorf("repo %q: %w", repo.ID, err)
		}
		desired, err := RenderREADME(template, RenderData{
			RepoID:            repo.ID,
			RepoUID:           repo.DurableID(),
			CanonicalProvider: repo.Canonical.Provider,
			CanonicalPath:     repo.Canonical.Path,
			Values:            readmeConfig.Values,
		})
		if err != nil {
			return Plan{}, fmt.Errorf("repo %q: render README: %w", repo.ID, err)
		}

		observed, err := observer.ObserveREADME(repo)
		if err != nil {
			return Plan{}, fmt.Errorf("repo %q: observe README: %w", repo.ID, err)
		}
		if err := validateObservation(observed); err != nil {
			return Plan{}, fmt.Errorf("repo %q: %w", repo.ID, err)
		}
		if observed.Present && bytes.Equal(observed.Content, desired) {
			continue
		}

		diff, err := ReviewDiff(observed.Present, observed.Content, desired)
		if err != nil {
			return Plan{}, fmt.Errorf("repo %q: build README review diff: %w", repo.ID, err)
		}
		desiredMode := "100644"
		if observed.Present {
			desiredMode = observed.Mode
		}
		desiredText := string(desired)
		present := observed.Present
		observedState := ObservedState{Present: &present}
		if observed.Present {
			observedState.Mode = observed.Mode
			observedState.SHA256 = DigestSHA256(observed.Content)
		}
		plan.Repositories = append(plan.Repositories, RepositoryPlan{
			UID: repo.DurableID(),
			ID:  repo.ID,
			Target: Target{
				Provider: repo.Canonical.Provider,
				Path:     repo.Canonical.Path,
				Branch:   observed.Branch,
			},
			BaseOID: observed.BaseOID,
			Actions: []Action{
				{
					Type:     ActionWriteREADME,
					Path:     READMEPath,
					Observed: observedState,
					Desired: DesiredState{
						Mode:    desiredMode,
						SHA256:  DigestSHA256(desired),
						Content: &desiredText,
					},
					TemplateSHA256: DigestSHA256(template),
					Diff:           diff,
				},
			},
		})
	}
	if err := plan.Validate(); err != nil {
		return Plan{}, fmt.Errorf("build managed artifact plan: %w", err)
	}
	return plan, nil
}

func validatePlannerRepositories(repos []config.Repo) error {
	seenUIDs := make(map[string]struct{}, len(repos))
	seenIDs := make(map[string]struct{}, len(repos))
	seenCanonical := make(map[string]struct{}, len(repos))
	for i, repo := range repos {
		uid := repo.DurableID()
		if !validPlanIdentifier(repo.ID) || !validPlanIdentifier(uid) {
			return fmt.Errorf("managed artifact repository %d requires canonical plan-safe id and uid", i)
		}
		if !validPlanIdentifier(repo.Canonical.Provider) {
			return fmt.Errorf("managed artifact repository %q canonical provider must be a plan-safe identifier", repo.ID)
		}
		if err := validatePlanProviderPath(repo.Canonical.Path); err != nil {
			return fmt.Errorf("managed artifact repository %q canonical path: %w", repo.ID, err)
		}
		if repo.Canonical.URL != "" {
			return fmt.Errorf("managed artifact repository %q must use provider/path canonical identity without legacy URL", repo.ID)
		}
		if _, exists := seenUIDs[uid]; exists {
			return fmt.Errorf("duplicate managed artifact repository uid %q", uid)
		}
		seenUIDs[uid] = struct{}{}
		if _, exists := seenIDs[repo.ID]; exists {
			return fmt.Errorf("duplicate managed artifact repository id %q", repo.ID)
		}
		seenIDs[repo.ID] = struct{}{}
		key := repo.Canonical.Provider + ":" + repo.Canonical.Path
		if _, exists := seenCanonical[key]; exists {
			return fmt.Errorf("duplicate managed artifact canonical repository %q", key)
		}
		seenCanonical[key] = struct{}{}
	}
	return nil
}

func validateObservation(observed READMEObservation) error {
	if err := validatePlanBranch(observed.Branch); err != nil {
		return fmt.Errorf("observed canonical branch: %w", err)
	}
	if !planOIDPattern.MatchString(observed.BaseOID) {
		return fmt.Errorf("observed canonical base OID must be a 40- or 64-character hexadecimal object ID")
	}
	if !observed.Present {
		if observed.Mode != "" || len(observed.Content) != 0 {
			return fmt.Errorf("absent README observation must not include mode or content")
		}
		return nil
	}
	if !validGitMode(observed.Mode) {
		return fmt.Errorf("observed README must be a regular Git blob with mode 100644 or 100755")
	}
	if len(observed.Content) > MaxTextBytes {
		return fmt.Errorf("observed README exceeds %d-byte limit", MaxTextBytes)
	}
	if err := validateManagedText("observed README content", string(observed.Content), true); err != nil {
		return err
	}
	return nil
}
