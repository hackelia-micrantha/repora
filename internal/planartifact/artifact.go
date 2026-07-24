// Package planartifact defines the durable, versioned representation of
// reconciliation plans. It deliberately excludes transport URLs and local paths.
package planartifact

import (
	"encoding/json"
	"fmt"
	"strings"

	"repoctl/internal/plan"
)

const (
	Version = 1
	Kind    = "repora.io/reconciliation-plan"
)

type Artifact struct {
	Version      int          `json:"version"`
	Kind         string       `json:"kind"`
	Repositories []Repository `json:"repositories"`
}

type Repository struct {
	UID     string   `json:"uid"`
	ID      string   `json:"id"`
	Actions []Action `json:"actions"`
}

type Action struct {
	Type   string `json:"type"`
	Source Ref    `json:"source"`
	Target Ref    `json:"target"`
	Diff   RefDiff `json:"diff"`
	Force  bool   `json:"force"`
	Reason string `json:"reason"`
}

type Ref struct {
	Provider string `json:"provider"`
	Remote   string `json:"remote"`
	Branch   string `json:"branch"`
}

type RefDiff struct {
	Observed string `json:"observed"`
	Desired  string `json:"desired"`
}

func FromPlans(plans ...plan.ReconciliationPlan) Artifact {
	artifact := Artifact{Version: Version, Kind: Kind, Repositories: make([]Repository, 0, len(plans))}
	for _, planned := range plans {
		repo := Repository{UID: planned.UID, ID: planned.ID, Actions: make([]Action, 0, len(planned.Actions))}
		for _, action := range planned.Actions {
			repo.Actions = append(repo.Actions, Action{
				Type: string(action.Type),
				Source: Ref{Provider: action.Source.Provider, Remote: action.Source.Name, Branch: action.Source.Branch},
				Target: Ref{Provider: action.Target.Provider, Remote: action.Target.Name, Branch: action.Target.Branch},
				Diff: RefDiff{Observed: action.ExpectedOldTarget, Desired: action.ExpectedSource},
				Force: action.Force,
				Reason: action.Reason,
			})
		}
		artifact.Repositories = append(artifact.Repositories, repo)
	}
	return artifact
}

func (a Artifact) Plans() ([]plan.ReconciliationPlan, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	plans := make([]plan.ReconciliationPlan, 0, len(a.Repositories))
	for _, repo := range a.Repositories {
		planned := plan.ReconciliationPlan{UID: repo.UID, ID: repo.ID, Actions: make([]plan.PlannedAction, 0, len(repo.Actions))}
		for _, action := range repo.Actions {
			planned.Actions = append(planned.Actions, plan.PlannedAction{
				Type: plan.ActionType(action.Type),
				Source: plan.Remote{Provider: action.Source.Provider, Name: action.Source.Remote, Branch: action.Source.Branch},
				Target: plan.Remote{Provider: action.Target.Provider, Name: action.Target.Remote, Branch: action.Target.Branch},
				ExpectedSource: action.Diff.Desired,
				ExpectedOldTarget: action.Diff.Observed,
				Force: action.Force,
				Reason: action.Reason,
			})
		}
		plans = append(plans, planned)
	}
	return plans, nil
}

func (a Artifact) Marshal() ([]byte, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return json.MarshalIndent(a, "", "  ")
}

func Parse(data []byte) (Artifact, error) {
	var artifact Artifact
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&artifact); err != nil {
		return Artifact{}, fmt.Errorf("decode plan artifact: %w", err)
	}
	if err := artifact.Validate(); err != nil {
		return Artifact{}, err
	}
	return artifact, nil
}

func (a Artifact) Validate() error {
	if a.Version != Version {
		return fmt.Errorf("unsupported plan artifact version %d", a.Version)
	}
	if a.Kind != Kind {
		return fmt.Errorf("unsupported plan artifact kind %q", a.Kind)
	}
	for i, repo := range a.Repositories {
		if strings.TrimSpace(repo.UID) == "" || strings.TrimSpace(repo.ID) == "" {
			return fmt.Errorf("repository %d requires uid and id", i)
		}
		if unsafeValue(repo.UID) || unsafeValue(repo.ID) {
			return fmt.Errorf("repository %d identity contains unsafe serialized data", i)
		}
		for j, action := range repo.Actions {
			if action.Type != string(plan.ActionPushBranch) {
				return fmt.Errorf("repository %d action %d has unsupported type %q", i, j, action.Type)
			}
			if err := validateRef(action.Source); err != nil {
				return fmt.Errorf("repository %d action %d source: %w", i, j, err)
			}
			if err := validateRef(action.Target); err != nil {
				return fmt.Errorf("repository %d action %d target: %w", i, j, err)
			}
			if strings.TrimSpace(action.Diff.Observed) == "" || strings.TrimSpace(action.Diff.Desired) == "" {
				return fmt.Errorf("repository %d action %d requires observed and desired refs", i, j)
			}
			if unsafeValue(action.Reason) {
				return fmt.Errorf("repository %d action %d reason contains unsafe serialized data", i, j)
			}
		}
	}
	return nil
}

func validateRef(ref Ref) error {
	if strings.TrimSpace(ref.Provider) == "" || strings.TrimSpace(ref.Remote) == "" || strings.TrimSpace(ref.Branch) == "" {
		return fmt.Errorf("provider, remote, and branch are required")
	}
	if unsafeValue(ref.Provider) || unsafeValue(ref.Remote) || unsafeValue(ref.Branch) {
		return fmt.Errorf("contains URL, credential, or local-path data")
	}
	return nil
}

func unsafeValue(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(value, "://") || strings.Contains(value, "token=") || strings.Contains(value, "password=") || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "file:")
}
