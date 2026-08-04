//go:build ignore

// route-tests validates the declarative document router against deterministic fixtures.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type budget struct {
	MaxFiles      int `yaml:"max_files" json:"max_files"`
	MaxBytes      int `yaml:"max_bytes" json:"max_bytes"`
	MaxTokensHint int `yaml:"max_tokens_hint" json:"max_tokens_hint"`
}

type matcher struct {
	AnyOf []string `yaml:"any_of"`
}

type route struct {
	ID       string   `yaml:"id"`
	Class    string   `yaml:"class"`
	Priority int      `yaml:"priority"`
	When     matcher  `yaml:"when"`
	Include  []string `yaml:"include"`
	Exclude  []string `yaml:"exclude"`
	Budget   budget   `yaml:"budget"`
}

type fallback struct {
	ID      string   `yaml:"id"`
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
	Budget  budget   `yaml:"budget"`
}

type router struct {
	Version   int        `yaml:"version"`
	Kind      string     `yaml:"kind"`
	Routes    []route    `yaml:"routes"`
	Fallbacks []fallback `yaml:"fallbacks"`
}

type fixtureFile struct {
	Version int       `json:"version"`
	Kind    string    `json:"kind"`
	Cases   []fixture `json:"cases"`
}

type fixture struct {
	Name           string   `json:"name"`
	Query          string   `json:"query"`
	ExpectRoutes   []string `json:"expect_routes"`
	ExpectFallback string   `json:"expect_fallback,omitempty"`
	ExpectInclude  []string `json:"expect_include,omitempty"`
	ExpectExclude  []string `json:"expect_exclude,omitempty"`
	ExpectBudget   *budget  `json:"expect_budget,omitempty"`
}

func main() {
	if len(os.Args) != 3 {
		fatalf("usage: go run ./scripts/ci/route-tests.go <router.yaml> <fixtures.json>")
	}

	var r router
	readYAML(os.Args[1], &r)
	if r.Version != 1 || r.Kind != "document-router" {
		fatalf("unsupported router contract: version=%d kind=%q", r.Version, r.Kind)
	}
	if len(r.Fallbacks) == 0 {
		fatalf("router must define at least one fallback")
	}

	var fixtures fixtureFile
	readJSON(os.Args[2], &fixtures)
	if fixtures.Version != 1 || fixtures.Kind != "document-route-tests" {
		fatalf("unsupported fixture contract: version=%d kind=%q", fixtures.Version, fixtures.Kind)
	}
	if len(fixtures.Cases) == 0 {
		fatalf("fixture file contains no cases")
	}

	seen := map[string]struct{}{}
	for _, tc := range fixtures.Cases {
		if tc.Name == "" {
			fatalf("fixture name must not be empty")
		}
		if _, ok := seen[tc.Name]; ok {
			fatalf("duplicate fixture name %q", tc.Name)
		}
		seen[tc.Name] = struct{}{}
		if err := validateCase(r, tc); err != nil {
			fatalf("%s: %v", tc.Name, err)
		}
		fmt.Printf("ok: %s\n", tc.Name)
	}
	fmt.Printf("validated %d deterministic route fixtures\n", len(fixtures.Cases))
}

func validateCase(r router, tc fixture) error {
	matched := selectRoutes(r.Routes, tc.Query)
	actualIDs := make([]string, len(matched))
	for i := range matched {
		actualIDs[i] = matched[i].ID
	}
	if !equalStrings(actualIDs, tc.ExpectRoutes) {
		return fmt.Errorf("routes: got %v, want %v", actualIDs, tc.ExpectRoutes)
	}

	if len(matched) == 0 {
		fb := r.Fallbacks[0]
		if tc.ExpectFallback != fb.ID {
			return fmt.Errorf("fallback: got %q, want %q", fb.ID, tc.ExpectFallback)
		}
		return validateSelection(fb.Include, fb.Exclude, fb.Budget, tc)
	}
	if tc.ExpectFallback != "" {
		return errors.New("fixture expects fallback despite matched routes")
	}
	return validateSelection(matched[0].Include, matched[0].Exclude, matched[0].Budget, tc)
}

func validateSelection(include, exclude []string, b budget, tc fixture) error {
	if len(tc.ExpectInclude) > 0 && !equalStrings(include, tc.ExpectInclude) {
		return fmt.Errorf("include: got %v, want %v", include, tc.ExpectInclude)
	}
	if len(tc.ExpectExclude) > 0 && !equalStrings(exclude, tc.ExpectExclude) {
		return fmt.Errorf("exclude: got %v, want %v", exclude, tc.ExpectExclude)
	}
	if tc.ExpectBudget != nil && b != *tc.ExpectBudget {
		return fmt.Errorf("budget: got %+v, want %+v", b, *tc.ExpectBudget)
	}
	return nil
}

func selectRoutes(routes []route, query string) []route {
	normalized := strings.ToLower(strings.Join(strings.Fields(query), " "))
	var matched []route
	for _, candidate := range routes {
		for _, term := range candidate.When.AnyOf {
			term = strings.ToLower(strings.Join(strings.Fields(term), " "))
			if term != "" && strings.Contains(normalized, term) {
				matched = append(matched, candidate)
				break
			}
		}
	}
	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].Priority != matched[j].Priority {
			return matched[i].Priority > matched[j].Priority
		}
		return matched[i].ID < matched[j].ID
	})
	return matched
}

func readYAML(path string, target any) {
	data, err := os.ReadFile(path)
	if err != nil {
		fatalf("read %s: %v", path, err)
	}
	if err := yaml.Unmarshal(data, target); err != nil {
		fatalf("parse %s: %v", path, err)
	}
}

func readJSON(path string, target any) {
	data, err := os.ReadFile(path)
	if err != nil {
		fatalf("read %s: %v", path, err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		fatalf("parse %s: %v", path, err)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "route-tests: "+format+"\n", args...)
	os.Exit(1)
}
