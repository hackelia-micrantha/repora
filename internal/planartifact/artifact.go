// Package planartifact defines the durable, versioned representation of
// reconciliation plans. It deliberately excludes transport URLs and local paths.
package planartifact

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"repoctl/internal/plan"
)

const (
	LegacyVersion = 1
	Version       = 2
	Kind          = "repora.io/reconciliation-plan"
)

var (
	identifierPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	oidPattern        = regexp.MustCompile(`^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`)
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
	Type   string  `json:"type"`
	Source Ref     `json:"source"`
	Target Ref     `json:"target"`
	Diff   RefDiff `json:"diff"`
	Force  bool    `json:"force"`
	Reason string  `json:"reason"`
}

type Ref struct {
	Provider string `json:"provider"`
	Path     string `json:"path,omitempty"`
	Remote   string `json:"remote"`
	Branch   string `json:"branch"`
}

type RefDiff struct {
	Observed string `json:"observed"`
	Desired  string `json:"desired"`
}

func SupportedVersion(version int) bool {
	return version == LegacyVersion || version == Version
}

// FromPlans preserves compatibility for focused internal callers: plans with
// complete provider paths emit v2, while historical alias-only plans emit v1.
// Production observation-to-artifact paths must use FromCurrentPlans.
func FromPlans(plans ...plan.ReconciliationPlan) Artifact {
	version := Version
	for _, planned := range plans {
		for _, action := range planned.Actions {
			if strings.TrimSpace(action.Source.Path) == "" || strings.TrimSpace(action.Target.Path) == "" {
				version = LegacyVersion
				break
			}
		}
		if version == LegacyVersion {
			break
		}
	}
	return fromPlans(version, plans...)
}

// FromCurrentPlans creates and validates a provider/path-bound v2 artifact.
func FromCurrentPlans(plans ...plan.ReconciliationPlan) (Artifact, error) {
	artifact := fromPlans(Version, plans...)
	if err := artifact.Validate(); err != nil {
		return Artifact{}, err
	}
	return artifact, nil
}

func FromLegacyPlans(plans ...plan.ReconciliationPlan) Artifact {
	return fromPlans(LegacyVersion, plans...)
}

func fromPlans(version int, plans ...plan.ReconciliationPlan) Artifact {
	artifact := Artifact{Version: version, Kind: Kind, Repositories: make([]Repository, 0, len(plans))}
	for _, planned := range plans {
		repo := Repository{UID: planned.UID, ID: planned.ID, Actions: make([]Action, 0, len(planned.Actions))}
		for _, action := range planned.Actions {
			source := Ref{Provider: action.Source.Provider, Remote: action.Source.Name, Branch: action.Source.Branch}
			target := Ref{Provider: action.Target.Provider, Remote: action.Target.Name, Branch: action.Target.Branch}
			if version == Version {
				source.Path = action.Source.Path
				target.Path = action.Target.Path
			}
			repo.Actions = append(repo.Actions, Action{
				Type:   string(action.Type),
				Source: source,
				Target: target,
				Diff:   RefDiff{Observed: action.ExpectedOldTarget, Desired: action.ExpectedSource},
				Force:  action.Force,
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
				Type:              plan.ActionType(action.Type),
				Source:            plan.Remote{Provider: action.Source.Provider, Path: action.Source.Path, Name: action.Source.Remote, Branch: action.Source.Branch},
				Target:            plan.Remote{Provider: action.Target.Provider, Path: action.Target.Path, Name: action.Target.Remote, Branch: action.Target.Branch},
				ExpectedSource:    action.Diff.Desired,
				ExpectedOldTarget: action.Diff.Observed,
				Force:             action.Force,
				Reason:            action.Reason,
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
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Artifact{}, fmt.Errorf("decode plan artifact: trailing JSON value")
		}
		return Artifact{}, fmt.Errorf("decode plan artifact: trailing data: %w", err)
	}
	if err := artifact.Validate(); err != nil {
		return Artifact{}, err
	}
	return artifact, nil
}

func (a Artifact) Validate() error {
	if !SupportedVersion(a.Version) {
		return fmt.Errorf("unsupported plan artifact version %d", a.Version)
	}
	if a.Kind != Kind {
		return fmt.Errorf("unsupported plan artifact kind %q", a.Kind)
	}
	for i, repo := range a.Repositories {
		if !validIdentifier(repo.UID) || !validIdentifier(repo.ID) {
			return fmt.Errorf("repository %d requires valid uid and id", i)
		}
		for j, action := range repo.Actions {
			if action.Type != string(plan.ActionPushBranch) {
				return fmt.Errorf("repository %d action %d has unsupported type %q", i, j, action.Type)
			}
			if err := validateRef(a.Version, action.Source); err != nil {
				return fmt.Errorf("repository %d action %d source: %w", i, j, err)
			}
			if err := validateRef(a.Version, action.Target); err != nil {
				return fmt.Errorf("repository %d action %d target: %w", i, j, err)
			}
			if !oidPattern.MatchString(strings.TrimSpace(action.Diff.Observed)) || !oidPattern.MatchString(strings.TrimSpace(action.Diff.Desired)) {
				return fmt.Errorf("repository %d action %d requires 40- or 64-character hexadecimal observed and desired object IDs", i, j)
			}
			if unsafeValue(action.Reason) {
				return fmt.Errorf("repository %d action %d reason contains unsafe serialized data", i, j)
			}
		}
	}
	return nil
}

func validateRef(version int, ref Ref) error {
	if !validIdentifier(ref.Provider) || !validIdentifier(ref.Remote) {
		return fmt.Errorf("provider and remote must be symbolic identifiers")
	}
	if version == LegacyVersion {
		if strings.TrimSpace(ref.Path) != "" {
			return fmt.Errorf("version 1 ref must not define provider path")
		}
	} else if err := validateProviderPath(ref.Path); err != nil {
		return err
	}
	return validateBranch(ref.Branch)
}

func validateProviderPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" || strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") {
		return fmt.Errorf("provider path contains an unsafe segment")
	}
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return fmt.Errorf("provider path must include an owner or namespace")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `\\:@?#`) || strings.ContainsAny(part, " \t\r\n") {
			return fmt.Errorf("provider path contains an unsafe segment")
		}
	}
	return nil
}

func validIdentifier(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && identifierPattern.MatchString(value)
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
	return strings.Contains(value, "://") || strings.Contains(value, "token=") || strings.Contains(value, "password=") || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "file:") || strings.Contains(value, "@")
}
