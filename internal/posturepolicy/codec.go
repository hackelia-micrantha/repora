package posturepolicy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"repoctl/internal/posture"
)

func ParseProfile(data []byte) (Profile, error) {
	var profile Profile
	if err := decodeStrict(data, &profile); err != nil {
		return Profile{}, fmt.Errorf("parse posture policy profile: %w", err)
	}
	if err := profile.Validate(); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

func ParseInputs(data []byte) (Inputs, error) {
	var inputs Inputs
	if err := decodeStrict(data, &inputs); err != nil {
		return Inputs{}, fmt.Errorf("parse posture policy inputs: %w", err)
	}
	if err := inputs.Validate(); err != nil {
		return Inputs{}, err
	}
	return inputs, nil
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

func NewInputs(repository string) Inputs {
	return Inputs{
		Kind:       InputsKind,
		Version:    InputsVersion,
		Repository: repository,
		Facts:      map[string]FactInput{},
	}
}

func (i Inputs) Validate() error {
	if i.Kind != InputsKind || i.Version != InputsVersion {
		return fmt.Errorf("unsupported posture policy inputs: kind=%q version=%d", i.Kind, i.Version)
	}
	if strings.TrimSpace(i.Repository) == "" {
		return fmt.Errorf("repository identity is required")
	}
	if i.Facts == nil {
		return fmt.Errorf("facts map is required")
	}
	keys := make([]string, 0, len(i.Facts))
	for key := range i.Facts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("fact name must not be empty")
		}
		if err := validateFactInput(key, i.Facts[key]); err != nil {
			return err
		}
	}
	return nil
}

func validateFactInput(name string, fact FactInput) error {
	if fact.Evidence == nil {
		return fmt.Errorf("fact %q evidence array is required", name)
	}
	switch fact.State {
	case posture.StateObserved:
		if len(fact.Value) == 0 || !json.Valid(fact.Value) {
			return fmt.Errorf("observed fact %q requires a valid JSON value", name)
		}
	case posture.StateUnknown, posture.StateUnavailable:
		if len(fact.Value) != 0 {
			return fmt.Errorf("fact %q in state %q must not carry a value", name, fact.State)
		}
	default:
		return fmt.Errorf("fact %q has unsupported state %q", name, fact.State)
	}
	for idx, evidence := range fact.Evidence {
		if strings.TrimSpace(evidence.Source) == "" || strings.TrimSpace(evidence.Reference) == "" {
			return fmt.Errorf("fact %q evidence[%d] requires source and reference", name, idx)
		}
	}
	return nil
}

func (p Profile) Marshal() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return marshalIndented(p, "posture policy profile")
}

func (i Inputs) Marshal() ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	return marshalIndented(i, "posture policy inputs")
}

func (r Report) Marshal() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return marshalIndented(r, "posture report")
}

func (r Report) Validate() error {
	if r.Kind != ReportKind || r.Version != ReportVersion {
		return fmt.Errorf("unsupported posture report: kind=%q version=%d", r.Kind, r.Version)
	}
	if strings.TrimSpace(r.Repository) == "" || strings.TrimSpace(r.ProfileID) == "" {
		return fmt.Errorf("posture report repository and profile_id are required")
	}
	if _, err := time.Parse("2006-01-02", r.AsOf); err != nil {
		return fmt.Errorf("posture report as_of must use YYYY-MM-DD: %w", err)
	}
	if r.Evaluations == nil {
		return fmt.Errorf("posture report evaluations array is required")
	}
	seen := map[string]struct{}{}
	for idx, evaluation := range r.Evaluations {
		if strings.TrimSpace(evaluation.RuleID) == "" || strings.TrimSpace(evaluation.Area) == "" || strings.TrimSpace(evaluation.Fact) == "" || strings.TrimSpace(evaluation.Title) == "" {
			return fmt.Errorf("evaluation[%d] requires rule_id, area, fact, and title", idx)
		}
		if _, exists := seen[evaluation.RuleID]; exists {
			return fmt.Errorf("duplicate evaluation rule_id %q", evaluation.RuleID)
		}
		seen[evaluation.RuleID] = struct{}{}
		if evaluation.Evidence == nil || evaluation.Remediation == nil {
			return fmt.Errorf("evaluation[%d] evidence and remediation arrays are required", idx)
		}
		switch evaluation.Status {
		case StatusPass, StatusFail, StatusWarning, StatusExcepted, StatusUnknown, StatusUnavailable:
		default:
			return fmt.Errorf("evaluation[%d] has unsupported status %q", idx, evaluation.Status)
		}
	}
	return nil
}

func marshalIndented(value any, name string) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", name, err)
	}
	return append(data, '\n'), nil
}
