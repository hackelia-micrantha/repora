package posturepolicy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"repoctl/internal/posture"
)

func Evaluate(profile Profile, inputs Inputs, asOf time.Time) (Report, error) {
	if err := profile.Validate(); err != nil {
		return Report{}, err
	}
	if err := inputs.Validate(); err != nil {
		return Report{}, err
	}

	exceptions := make(map[string]Exception, len(profile.Exceptions))
	for _, exception := range profile.Exceptions {
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
		if exception, ok := exceptions[rule.ID]; ok && isMismatch(evaluation.Status) {
			expires, _ := time.Parse("2006-01-02", exception.Expires)
			evaluation.Exception = &exception
			if asOf.UTC().Before(expires.Add(24 * time.Hour)) {
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
		AsOf:        asOf.UTC().Format("2006-01-02"),
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
		Evidence:    append([]posture.Evidence{}, fact.Evidence...),
		Remediation: append([]string{}, rule.Remediation...),
	}

	switch fact.State {
	case posture.StateUnknown:
		evaluation.Status = StatusUnknown
		return evaluation, nil
	case posture.StateUnavailable:
		evaluation.Status = StatusUnavailable
		return evaluation, nil
	case posture.StateObserved:
		evaluation.Observed = cloneRaw(fact.Value)
	default:
		return Evaluation{}, fmt.Errorf("fact %q has unsupported state %q", rule.Fact, fact.State)
	}

	matched, err := matches(rule, fact.Value)
	if err != nil {
		return Evaluation{}, err
	}
	if matched {
		evaluation.Status = StatusPass
	} else if rule.Severity == SeverityInfo {
		evaluation.Status = StatusWarning
	} else {
		evaluation.Status = StatusFail
	}
	return evaluation, nil
}

func isMismatch(status ResultStatus) bool {
	return status == StatusFail || status == StatusWarning
}

func matches(rule Rule, observed json.RawMessage) (bool, error) {
	switch rule.Operator {
	case OperatorEquals:
		left, err := decodeJSONValue(observed)
		if err != nil {
			return false, fmt.Errorf("decode observed value: %w", err)
		}
		right, err := decodeJSONValue(rule.Expected)
		if err != nil {
			return false, fmt.Errorf("decode expected value: %w", err)
		}
		return equalJSONValue(left, right), nil
	case OperatorAtLeast, OperatorAtMost:
		comparison, err := compareJSONNumbers(observed, rule.Expected)
		if err != nil {
			return false, err
		}
		if rule.Operator == OperatorAtLeast {
			return comparison >= 0, nil
		}
		return comparison <= 0, nil
	case OperatorNonEmpty:
		value, err := decodeJSONValue(observed)
		if err != nil {
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

func decodeJSONValue(raw json.RawMessage) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func equalJSONValue(left, right any) bool {
	switch l := left.(type) {
	case json.Number:
		r, ok := right.(json.Number)
		if !ok {
			return false
		}
		comparison, err := compareNumberStrings(l.String(), r.String())
		return err == nil && comparison == 0
	case []any:
		r, ok := right.([]any)
		if !ok || len(l) != len(r) {
			return false
		}
		for idx := range l {
			if !equalJSONValue(l[idx], r[idx]) {
				return false
			}
		}
		return true
	case map[string]any:
		r, ok := right.(map[string]any)
		if !ok || len(l) != len(r) {
			return false
		}
		for key, value := range l {
			other, exists := r[key]
			if !exists || !equalJSONValue(value, other) {
				return false
			}
		}
		return true
	case string:
		r, ok := right.(string)
		return ok && l == r
	case bool:
		r, ok := right.(bool)
		return ok && l == r
	case nil:
		return right == nil
	default:
		return false
	}
}

func compareJSONNumbers(observed, expected json.RawMessage) (int, error) {
	left, err := decodeJSONValue(observed)
	if err != nil {
		return 0, fmt.Errorf("observed value: %w", err)
	}
	right, err := decodeJSONValue(expected)
	if err != nil {
		return 0, fmt.Errorf("expected value: %w", err)
	}
	leftNumber, ok := left.(json.Number)
	if !ok {
		return 0, fmt.Errorf("observed value: must be numeric")
	}
	rightNumber, ok := right.(json.Number)
	if !ok {
		return 0, fmt.Errorf("expected value: must be numeric")
	}
	comparison, err := compareNumberStrings(leftNumber.String(), rightNumber.String())
	if err != nil {
		return 0, fmt.Errorf("compare numeric values: %w", err)
	}
	return comparison, nil
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
		if isMismatch(evaluation.Status) {
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
