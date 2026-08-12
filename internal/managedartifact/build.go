package managedartifact

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

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
	if observer == nil {
		return Plan{}, fmt.Errorf("README observer is required")
	}

	repos := make([]config.Repo, 0, len(spec.Repos))
	for _, repo := range spec.Repos {
		if repo.Artifacts.Readme != nil {
			repos = append(repos, repo)
		}
	}
	sort.Slice(repos, func(i, j int) bool {
		left, right := repos[i].DurableID(), repos[j].DurableID()
		if left != right {
			return left < right
		}
		return repos[i].ID < repos[j].ID
	})

	plan := Plan{Kind: PlanKind, Version: PlanVersion, Repositories: []RepositoryPlan{}}
	for _, repo := range repos {
		readmeConfig := repo.Artifacts.Readme
		template, err := LoadTemplate(configPath, readmeConfig.Template)
		if err != nil {
			return Plan{}, fmt.Errorf("repo %q: %w", repo.ID, err)
		}
		desired, err := RenderREADME(template, RenderData{
			RepoID:            repo.ID,
			RepoUID:           repo.DurableID(),
			CanonicalProvider: strings.TrimSpace(repo.Canonical.Provider),
			CanonicalPath:     strings.Trim(strings.TrimSpace(repo.Canonical.Path), "/"),
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

		diff, err := ReviewDiff(observedBytes(observed), desired)
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
		repositoryPlan := RepositoryPlan{
			UID: repo.DurableID(),
			ID:  repo.ID,
			Target: Target{
				Provider: strings.TrimSpace(repo.Canonical.Provider),
				Path:     strings.Trim(strings.TrimSpace(repo.Canonical.Path), "/"),
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
		}
		plan.Repositories = append(plan.Repositories, repositoryPlan)
	}
	if err := plan.Validate(); err != nil {
		return Plan{}, fmt.Errorf("build managed artifact plan: %w", err)
	}
	return plan, nil
}

func validateObservation(observed READMEObservation) error {
	if observed.Branch == "" {
		return fmt.Errorf("observed canonical branch is required")
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

func observedBytes(observed READMEObservation) []byte {
	if !observed.Present {
		return nil
	}
	return observed.Content
}
