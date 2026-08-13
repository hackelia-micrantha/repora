//go:build ignore

// route-tests validates the declarative document router against deterministic fixtures.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	ID           string   `yaml:"id"`
	Class        string   `yaml:"class"`
	Priority     int      `yaml:"priority"`
	When         matcher  `yaml:"when"`
	TrustInclude []string `yaml:"trust_include"`
	Include      []string `yaml:"include"`
	Exclude      []string `yaml:"exclude"`
	Budget       budget   `yaml:"budget"`
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
	Manifests []string   `yaml:"manifests"`
	Routes    []route    `yaml:"routes"`
	Fallbacks []fallback `yaml:"fallbacks"`
}

type manifest struct {
	Version int     `yaml:"version"`
	Kind    string  `yaml:"kind"`
	Owner   string  `yaml:"owner"`
	Routes  []route `yaml:"routes"`
}

type fixtureFile struct {
	Version int       `json:"version"`
	Kind    string    `json:"kind"`
	Cases   []fixture `json:"cases"`
}

type fixture struct {
	Name               string   `json:"name"`
	Query              string   `json:"query"`
	ExpectRoutes       []string `json:"expect_routes"`
	ExpectFallback     string   `json:"expect_fallback,omitempty"`
	ExpectTrustInclude []string `json:"expect_trust_include,omitempty"`
	ExpectInclude      []string `json:"expect_include,omitempty"`
	ExpectExclude      []string `json:"expect_exclude,omitempty"`
	ExpectBudget       *budget  `json:"expect_budget,omitempty"`
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

	r.Routes = composeRoutes(os.Args[1], r.Routes, r.Manifests)

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
	fmt.Printf("validated %d deterministic route fixtures across %d manifests\n", len(fixtures.Cases), len(r.Manifests))
}

func composeRoutes(routerPath string, rootRoutes []route, manifestPaths []string) []route {
	baseDir := filepath.Dir(routerPath)
	routes := append([]route(nil), rootRoutes...)
	seen := map[string]string{}
	for _, candidate := range routes {
		registerRoute(seen, candidate, "root router")
	}

	manifestSeen := map[string]struct{}{}
	for _, configuredPath := range manifestPaths {
		clean := filepath.Clean(configuredPath)
		if configuredPath == "" || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			fatalf("unsafe manifest path %q", configuredPath)
		}
		if _, ok := manifestSeen[clean]; ok {
			fatalf("duplicate manifest path %q", configuredPath)
		}
		manifestSeen[clean] = struct{}{}

		path := filepath.Join(baseDir, "..", clean)
		var m manifest
		readYAML(path, &m)
		if m.Version != 1 || m.Kind != "document-route-manifest" {
			fatalf("unsupported manifest %q: version=%d kind=%q", configuredPath, m.Version, m.Kind)
		}
		if strings.TrimSpace(m.Owner) == "" {
			fatalf("manifest %q must declare an owner", configuredPath)
		}
		if len(m.Routes) == 0 {
			fatalf("manifest %q contains no routes", configuredPath)
		}
		for _, candidate := range m.Routes {
			registerRoute(seen, candidate, configuredPath)
			routes = append(routes, candidate)
		}
	}
	return routes
}

func registerRoute(seen map[string]string, candidate route, source string) {
	if strings.TrimSpace(candidate.ID) == "" {
		fatalf("route in %s has an empty id", source)
	}
	if previous, ok := seen[candidate.ID]; ok {
		fatalf("duplicate route id %q in %s and %s", candidate.ID, previous, source)
	}
	seen[candidate.ID] = source
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
		return validateSelection(nil, fb.Include, fb.Exclude, fb.Budget, tc)
	}
	if tc.ExpectFallback != "" {
		return errors.New("fixture expects fallback despite matched routes")
	}
	return validateSelection(matched[0].TrustInclude, matched[0].Include, matched[0].Exclude, matched[0].Budget, tc)
}

func validateSelection(trustInclude, include, exclude []string, b budget, tc fixture) error {
	if !equalStrings(trustInclude, tc.ExpectTrustInclude) {
		return fmt.Errorf("trust_include: got %v, want %v", trustInclude, tc.ExpectTrustInclude)
	}
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
			if term != "" && containsTerm(normalized, term) {
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

func containsTerm(query, term string) bool {
	for start := 0; start <= len(query)-len(term); {
		offset := strings.Index(query[start:], term)
		if offset < 0 {
			return false
		}
		left := start + offset
		right := left + len(term)
		if (left == 0 || !isWordByte(query[left-1])) && (right == len(query) || !isWordByte(query[right])) {
			return true
		}
		start = left + 1
	}
	return false
}

func isWordByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '_'
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
