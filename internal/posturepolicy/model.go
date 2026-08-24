// Package posturepolicy evaluates normalized posture facts without re-scanning repositories.
package posturepolicy

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"repoctl/internal/posture"
)

const (
	ProfileKind    = "repora.posture-policy-profile"
	ProfileVersion = 1
	ReportKind     = "repora.posture-report"
	ReportVersion  = 1
)

type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "informational"
)

type Operator string

const (
	OperatorEquals   Operator = "equals"
	OperatorAtLeast  Operator = "at_least"
	OperatorAtMost   Operator = "at_most"
	OperatorNonEmpty Operator = "non_empty"
)

type Profile struct {
	Kind       string      `json:"kind" yaml:"kind"`
	Version    int         `json:"version" yaml:"version"`
	ID         string      `json:"id" yaml:"id"`
	Rules      []Rule      `json:"rules" yaml:"rules"`
	Exceptions []Exception `json:"exceptions" yaml:"exceptions"`
}

type Rule struct {
	ID          string          `json:"id" yaml:"id"`
	Area        string          `json:"area" yaml:"area"`
	Fact        string          `json:"fact" yaml:"fact"`
	Operator    Operator        `json:"operator" yaml:"operator"`
	Expected    json.RawMessage `json:"expected,omitempty" yaml:"expected,omitempty"`
	Severity    Severity        `json:"severity" yaml:"severity"`
	Title       string          `json:"title" yaml:"title"`
	Remediation []string        `json:"remediation" yaml:"remediation"`
}

type Exception struct {
	RuleID  string `json:"rule_id" yaml:"rule_id"`
	Reason  string `json:"reason" yaml:"reason"`
	Owner   string `json:"owner" yaml:"owner"`
	Expires string `json:"expires" yaml:"expires"`
}

type FactInput struct {
	State    posture.FactState  `json:"state"`
	Value    json.RawMessage    `json:"value,omitempty"`
	Evidence []posture.Evidence `json:"evidence"`
}

type Inputs struct {
	Repository string               `json:"repository"`
	Facts      map[string]FactInput `json:"facts"`
}

type ResultStatus string

const (
	StatusPass        ResultStatus = "pass"
	StatusFail        ResultStatus = "fail"
	StatusExcepted    ResultStatus = "excepted"
	StatusUnknown     ResultStatus = "unknown"
	StatusUnavailable ResultStatus = "unavailable"
)

type Evaluation struct {
	RuleID       string             `json:"rule_id"`
	Area         string             `json:"area"`
	Fact         string             `json:"fact"`
	Severity     Severity           `json:"severity"`
	Status       ResultStatus       `json:"status"`
	Title        string             `json:"title"`
	Expected     json.RawMessage    `json:"expected,omitempty"`
	Observed     json.RawMessage    `json:"observed,omitempty"`
	Evidence     []posture.Evidence `json:"evidence"`
	Remediation  []string           `json:"remediation"`
	Exception    *Exception         `json:"exception,omitempty"`
	ExceptionGap string             `json:"exception_gap,omitempty"`
}

type Report struct {
	Kind        string       `json:"kind"`
	Version     int          `json:"version"`
	Repository  string       `json:"repository"`
	ProfileID   string       `json:"profile_id"`
	Evaluations []Evaluation `json:"evaluations"`
}

func (p Profile) Validate() error {
	if p.Kind != ProfileKind || p.Version != ProfileVersion {
		return fmt.Errorf("unsupported posture policy profile: kind=%q version=%d", p.Kind, p.Version)
	}
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("profile id is required")
	}
	if p.Rules == nil || p.Exceptions == nil {
		return fmt.Errorf("profile rules and exceptions arrays are required")
	}
	seen := map[string]struct{}{}
	for i, rule := range p.Rules {
		if err := rule.validate(); err != nil {
			return fmt.Errorf("rule[%d]: %w", i, err)
		}
		if _, exists := seen[rule.ID]; exists {
			return fmt.Errorf("duplicate rule id %q", rule.ID)
		}
		seen[rule.ID] = struct{}{}
	}
	for i, exception := range p.Exceptions {
		if _, exists := seen[exception.RuleID]; !exists {
			return fmt.Errorf("exception[%d] references unknown rule %q", i, exception.RuleID)
		}
		if strings.TrimSpace(exception.Reason) == "" || strings.TrimSpace(exception.Owner) == "" || strings.TrimSpace(exception.Expires) == "" {
			return fmt.Errorf("exception[%d] requires reason, owner, and expiry", i)
		}
		if _, err := time.Parse("2006-01-02", exception.Expires); err != nil {
			return fmt.Errorf("exception[%d] expiry must use YYYY-MM-DD: %w", i, err)
		}
	}
	return nil
}

func (r Rule) validate() error {
	if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.Area) == "" || strings.TrimSpace(r.Fact) == "" || strings.TrimSpace(r.Title) == "" {
		return fmt.Errorf("id, area, fact, and title are required")
	}
	switch r.Operator {
	case OperatorEquals, OperatorAtLeast, OperatorAtMost:
		if len(r.Expected) == 0 {
			return fmt.Errorf("operator %q requires expected value", r.Operator)
		}
	case OperatorNonEmpty:
		if len(r.Expected) != 0 {
			return fmt.Errorf("operator %q must not define expected value", r.Operator)
		}
	default:
		return fmt.Errorf("unsupported operator %q", r.Operator)
	}
	switch r.Severity {
	case SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo:
	default:
		return fmt.Errorf("unsupported severity %q", r.Severity)
	}
	if r.Remediation == nil {
		return fmt.Errorf("remediation array is required")
	}
	return nil
}

func sortedEvaluations(values []Evaluation) []Evaluation {
	out := append([]Evaluation(nil), values...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Area != out[j].Area {
			return out[i].Area < out[j].Area
		}
		return out[i].RuleID < out[j].RuleID
	})
	return out
}
