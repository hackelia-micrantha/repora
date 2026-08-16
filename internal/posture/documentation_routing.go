package posture

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

func parseTrustRouter(data []byte) (trustRouter, error) {
	var raw struct {
		Version int    `yaml:"version"`
		Kind    string `yaml:"kind"`
		Trust   struct {
			Rules []struct {
				Tier  string   `yaml:"tier"`
				Paths []string `yaml:"paths"`
			} `yaml:"rules"`
		} `yaml:"trust"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return trustRouter{}, fmt.Errorf("parse document router: %w", err)
	}
	if raw.Version != 1 || raw.Kind != "document-router" {
		return trustRouter{}, fmt.Errorf("unsupported document router contract: kind=%q version=%d", raw.Kind, raw.Version)
	}
	validTiers := map[string]bool{
		"canonical":      true,
		"implementation": true,
		"generated":      true,
		"experimental":   true,
		"archived":       true,
		"external":       true,
	}
	seenPatterns := map[string]struct{}{}
	router := trustRouter{Rules: []trustRule{}}
	for index, rawRule := range raw.Trust.Rules {
		if !validTiers[rawRule.Tier] {
			return trustRouter{}, fmt.Errorf("document router contains unknown trust tier %q", rawRule.Tier)
		}
		if len(rawRule.Paths) == 0 {
			return trustRouter{}, fmt.Errorf("document router trust tier %q has no paths", rawRule.Tier)
		}
		for _, pattern := range rawRule.Paths {
			if pattern == "" {
				return trustRouter{}, fmt.Errorf("document router contains an empty trust pattern")
			}
			if strings.ContainsAny(pattern, "[]") {
				return trustRouter{}, fmt.Errorf("document router trust pattern %q uses unsupported character-class syntax", pattern)
			}
			if _, exists := seenPatterns[pattern]; exists {
				return trustRouter{}, fmt.Errorf("document router contains duplicate trust pattern %q", pattern)
			}
			seenPatterns[pattern] = struct{}{}
		}
		router.Rules = append(router.Rules, trustRule{Tier: rawRule.Tier, Patterns: append([]string(nil), rawRule.Paths...), Index: index})
	}
	if len(router.Rules) == 0 {
		return trustRouter{}, fmt.Errorf("document router contains no trust rules")
	}
	return router, nil
}

func classifyTrustFact(documentPath string, router trustRouter, routerState FactState, evidence Evidence) Fact[string] {
	switch routerState {
	case StateUnavailable:
		return Unavailable[string](evidence)
	case StateObserved:
		tier, err := router.classify(documentPath)
		if err != nil {
			return Unknown[string](evidenceWithDetail(evidence, sanitizeDocumentationError(err)))
		}
		return Observed(tier, evidence)
	default:
		return Unknown[string](evidence)
	}
}

func (r trustRouter) classify(candidate string) (string, error) {
	type match struct {
		literal int
		length  int
		index   int
		tier    string
	}
	var best *match
	for _, rule := range r.Rules {
		for _, pattern := range rule.Patterns {
			matched, err := fnmatchCase(pattern, candidate)
			if err != nil {
				return "", err
			}
			if !matched {
				continue
			}
			literal := len(strings.NewReplacer("*", "", "?", "").Replace(pattern))
			current := match{literal: literal, length: len(pattern), index: rule.Index, tier: rule.Tier}
			if best == nil || current.literal > best.literal ||
				(current.literal == best.literal && current.length > best.length) ||
				(current.literal == best.literal && current.length == best.length && current.index < best.index) {
				copy := current
				best = &copy
			}
		}
	}
	if best == nil {
		return "unclassified", nil
	}
	return best.tier, nil
}

func fnmatchCase(pattern, candidate string) (bool, error) {
	var expression strings.Builder
	expression.WriteString("^")
	for _, r := range pattern {
		switch r {
		case '*':
			expression.WriteString(".*")
		case '?':
			expression.WriteString(".")
		case '[', ']':
			return false, fmt.Errorf("trust pattern %q uses unsupported character-class syntax", pattern)
		default:
			expression.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	expression.WriteString("$")
	compiled, err := regexp.Compile(expression.String())
	if err != nil {
		return false, fmt.Errorf("compile trust pattern %q: %w", pattern, err)
	}
	return compiled.MatchString(candidate), nil
}
