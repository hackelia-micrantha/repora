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
		var applicability *ApplicabilityEvaluation
		if rule.Applicability != nil {
			applicabilityFact := inputFact(inputs, rule.Applicability.Fact)
			applicabilityEvaluation, applies, status, err := evaluateApplicability(*rule.Applicability, applicabilityFact)
			if err != nil {
				return Report{}, fmt.Errorf("evaluate rule %q applicability: %w", rule.ID, err)
			}
			applicability = &applicabilityEvaluation
			if !applies {
				evaluation := newEvaluation(rule, applicabilityFact.Evidence)
				evaluation.Status = status
				evaluation.Applicability = applicability
				evaluations = append(evaluations, evaluation)
				continue
			}
		}

		fact := inputFact(inputs, rule.Fact)
		evaluation, err := evaluateRule(rule, fact)
		if err != nil {
			return Report{}, fmt.Errorf("evaluate rule %q: %w", rule.ID, err)
		}
		evaluation.Applicability = applicability
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

	reportVersion := ReportVersion
	if profile.Version == ProfileVersionV2 {
		reportVersion = ReportVersionV2
	}
	return Report{
		Kind:        ReportKind,
		Version:     reportVersion,
		Repository:  inputs.Repository,
		ProfileID:   profile.ID,
		AsOf:        asOf.UTC().Format("2006-01-02"),
		Evaluations: sortedEvaluations(evaluations),
	}, nil
}

func inputFact(inputs Inputs, name string) FactInput {
	if fact, exists := inputs.Facts[name]; exists {
		return fact
	}
	return FactInput{
		State: posture.StateUnknown,
		Evidence: []posture.Evidence{{
			Source:    "posture-policy",
			Reference: name,
			Detail:    "normalized fact was not supplied to the convergence layer",
		}},
	}
}

func evaluateApplicability(selector RuleApplicability, fact FactInput) (ApplicabilityEvaluation, bool, ResultStatus, error) {
	evaluation := ApplicabilityEvaluation{
		Fact:              selector.Fact,
		State:             fact.State,
		Evidence:          append([]posture.Evidence{}, fact.Evidence...),
		ApplicableWhen:    cloneCondition(selector.ApplicableWhen),
		NotApplicableWhen: cloneCondition(selector.NotApplicableWhen),
	}
	switch fact.State {
	case posture.StateUnknown:
		evaluation.Decision = ApplicabilityUnknown
		return evaluation, false, StatusUnknown, nil
	case posture.StateUnavailable:
		evaluation.Decision = ApplicabilityUnavailable
		return evaluation, false, StatusUnavailable, nil
	case posture.StateObserved:
		evaluation.Observed = cloneRaw(fact.Value)
	default:
		return ApplicabilityEvaluation{}, false, "", fmt.Errorf("fact %q has unsupported state %q", selector.Fact, fact.State)
	}

	applies, err := matchesCondition(selector.ApplicableWhen, fact.Value)
	if err != nil {
		return ApplicabilityEvaluation{}, false, "", fmt.Errorf("applicable_when: %w", err)
	}
	notApplicable, err := matchesCondition(selector.NotApplicableWhen, fact.Value)
	if err != nil {
		return ApplicabilityEvaluation{}, false, "", fmt.Errorf("not_applicable_when: %w", err)
	}
	if applies && notApplicable {
		return ApplicabilityEvaluation{}, false, "", fmt.Errorf("applicability fact %q matches both applicable_when and not_applicable_when", selector.Fact)
	}
	if applies {
		evaluation.Decision = ApplicabilityApplicable
		return evaluation, true, "", nil
	}
	if notApplicable {
		evaluation.Decision = ApplicabilityNotApplicable
		return evaluation, false, StatusNotApplicable, nil
	}
	evaluation.Decision = ApplicabilityUnresolved
	return evaluation, false, StatusUnknown, nil
}

func cloneCondition(condition Condition) Condition {
	return Condition{Operator: condition.Operator, Expected: cloneRaw(condition.Expected)}
}

func newEvaluation(rule Rule, evidence []posture.Evidence) Evaluation {
	return Evaluation{
		RuleID:      rule.ID,
		Area:        rule.Area,
		Fact:        rule.Fact,
		Severity:    rule.Severity,
		Title:       rule.Title,
		Expected:    cloneRaw(rule.Expected),
		Evidence:    append([]posture.Evidence{}, evidence...),
		Remediation: append([]string{}, rule.Remediation...),
	}
}

func evaluateRule(rule Rule, fact FactInput) (Evaluation, error) {
	evaluation := newEvaluation(rule, fact.Evidence)

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

	matched, err := matchesCondition(Condition{Operator: rule.Operator, Expected: rule.Expected}, fact.Value)
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

func matchesCondition(condition Condition, observed json.RawMessage) (bool, error) {
	switch condition.Operator {
	case OperatorEquals:
		left, err := decodeJSONValue(observed)
		if err != nil {
			return false, fmt.Errorf("decode observed value: %w", err)
		}
		right, err := decodeJSONValue(condition.Expected)
		if err != nil {
			return false, fmt.Errorf("decode expected value: %w", err)
		}
		return equalJSONValue(left, right), nil
	case OperatorAtLeast, OperatorAtMost:
		comparison, err := compareJSONNumbers(observed, condition.Expected)
		if err != nil {
			return false, err
		}
		if condition.Operator == OperatorAtLeast {
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
		return false, fmt.Errorf("unsupported operator %q", condition.Operator)
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
