package posturepolicy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"time"

	"repoctl/internal/posture"
)

func Evaluate(profile Profile, inputs Inputs, asOf time.Time) (Report, error) {
	if err := profile.Validate(); err != nil {
		return Report{}, err
	}
	if inputs.Repository == "" {
		return Report{}, fmt.Errorf("repository identity is required")
	}
	if inputs.Facts == nil {
		return Report{}, fmt.Errorf("facts map is required")
	}

	exceptions := make(map[string]Exception, len(profile.Exceptions))
	for _, exception := range profile.Exceptions {
		if _, exists := exceptions[exception.RuleID]; exists {
			return Report{}, fmt.Errorf("duplicate exception for rule %q", exception.RuleID)
		}
		exceptions[exception.RuleID] = exception
	}

	evaluations := make([]Evaluation, 0, len(profile.Rules))
	for _, rule := range profile.Rules {
		fact, exists := inputs.Facts[rule.Fact]
		if !exists {
			fact = FactInput{
				State: posture.StateUnknown,
				Evidence: []posture.Evidence{{
					Source:    "posture-policy",
					Reference: rule.Fact,
					Detail:    "normalized fact was not supplied to the convergence layer",
				}},
			}
		}
		evaluation, err := evaluateRule(rule, fact)
		if err != nil {
			return Report{}, fmt.Errorf("evaluate rule %q: %w", rule.ID, err)
		}
		if exception, ok := exceptions[rule.ID]; ok && evaluation.Status == StatusFail {
			expires, _ := time.Parse("2006-01-02", exception.Expires)
			evaluation.Exception = &exception
			if asOf.Before(expires.Add(24 * time.Hour)) {
				evaluation.Status = StatusExcepted
			} else {
				evaluation.ExceptionGap = "exception expired"
			}
		}
		evaluations = append(evaluations, evaluation)
	}

	return Report{
		Kind:        ReportKind,
		Version:     ReportVersion,
		Repository:  inputs.Repository,
		ProfileID:   profile.ID,
		Evaluations: sortedEvaluations(evaluations),
	}, nil
}

func evaluateRule(rule Rule, fact FactInput) (Evaluation, error) {
	evaluation := Evaluation{
		RuleID:      rule.ID,
		Area:        rule.Area,
		Fact:        rule.Fact,
		Severity:    rule.Severity,
		Title:       rule.Title,
		Expected:    cloneRaw(rule.Expected),
		Evidence:    append([]posture.Evidence(nil), fact.Evidence...),
		Remediation: append([]string(nil), rule.Remediation...),
	}

	switch fact.State {
	case posture.StateUnknown:
		evaluation.Status = StatusUnknown
		return evaluation, nil
	case posture.StateUnavailable:
		evaluation.Status = StatusUnavailable
		return evaluation, nil
	case posture.StateObserved:
		if len(fact.Value) == 0 {
			return Evaluation{}, fmt.Errorf("observed fact %q has no value", rule.Fact)
		}
		evaluation.Observed = cloneRaw(fact.Value)
	case "":
		return Evaluation{}, fmt.Errorf("fact %q has empty state", rule.Fact)
	default:
		return Evaluation{}, fmt.Errorf("fact %q has unsupported state %q", rule.Fact, fact.State)
	}

	matched, err := matches(rule, fact.Value)
	if err != nil {
		return Evaluation{}, err
	}
	if matched {
		evaluation.Status = StatusPass
	} else {
		evaluation.Status = StatusFail
	}
	return evaluation, nil
}

func matches(rule Rule, observed json.RawMessage) (bool, error) {
	switch rule.Operator {
	case OperatorEquals:
		var left, right any
		if err := json.Unmarshal(observed, &left); err != nil {
			return false, fmt.Errorf("decode observed value: %w", err)
		}
		if err := json.Unmarshal(rule.Expected, &right); err != nil {
			return false, fmt.Errorf("decode expected value: %w", err)
		}
		return reflect.DeepEqual(left, right), nil
	case OperatorAtLeast, OperatorAtMost:
		observedNumber, err := decodeNumber(observed)
		if err != nil {
			return false, fmt.Errorf("observed value: %w", err)
		}
		expectedNumber, err := decodeNumber(rule.Expected)
		if err != nil {
			return false, fmt.Errorf("expected value: %w", err)
		}
		if rule.Operator == OperatorAtLeast {
			return observedNumber >= expectedNumber, nil
		}
		return observedNumber <= expectedNumber, nil
	case OperatorNonEmpty:
		var value any
		if err := json.Unmarshal(observed, &value); err != nil {
			return false, fmt.Errorf("decode observed value: %w", err)
		}
		switch current := value.(type) {
		case string:
			return current != "", nil
		case []any:
			return len(current) > 0, nil
		case map[string]any:
			return len(current) > 0, nil
		default:
			return false, nil
		}
	default:
		return false, fmt.Errorf("unsupported operator %q", rule.Operator)
	}
}

func decodeNumber(raw json.RawMessage) (float64, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return 0, err
	}
	number, ok := value.(json.Number)
	if !ok {
		return 0, fmt.Errorf("must be numeric")
	}
	return number.Float64()
}

func cloneRaw(value json.RawMessage) json.RawMessage {
	if value == nil {
		return nil
	}
	return append(json.RawMessage(nil), value...)
}

func SummaryBySeverity(report Report) map[Severity]int {
	out := map[Severity]int{}
	for _, evaluation := range report.Evaluations {
		if evaluation.Status == StatusFail {
			out[evaluation.Severity]++
		}
	}
	return out
}

func Unsupported(report Report) []Evaluation {
	out := make([]Evaluation, 0)
	for _, evaluation := range report.Evaluations {
		if evaluation.Status == StatusUnknown || evaluation.Status == StatusUnavailable {
			out = append(out, evaluation)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RuleID < out[j].RuleID })
	return out
}
