//go:build ignore

// go-ast-routing validates deterministic Go source selectors against configured routes.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type sourceSelectors struct {
	Language string   `yaml:"language" json:"language"`
	Packages []string `yaml:"packages" json:"packages"`
	Symbols  []string `yaml:"symbols" json:"symbols"`
	Commands []string `yaml:"commands" json:"commands"`
	Files    []string `yaml:"files" json:"files"`
}

type route struct {
	ID              string           `yaml:"id"`
	Include         []string         `yaml:"include"`
	Exclude         []string         `yaml:"exclude"`
	SourceSelectors *sourceSelectors `yaml:"source_selectors"`
}

type router struct {
	Version   int      `yaml:"version"`
	Kind      string   `yaml:"kind"`
	Manifests []string `yaml:"manifests"`
	Routes    []route  `yaml:"routes"`
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
	Name          string              `json:"name"`
	RouteID       string              `json:"route_id"`
	ExpectPaths   []string            `json:"expect_paths"`
	ExpectReasons map[string][]string `json:"expect_reasons"`
}

type sourceFile struct {
	Path       string
	PackageDir string
	Command    string
	Symbols    map[string]struct{}
}

func main() {
	if len(os.Args) != 4 {
		fatalf("usage: go run ./scripts/ci/go-ast-routing.go <repo-root> <router.yaml> <fixtures.json>")
	}
	root, err := filepath.Abs(os.Args[1])
	if err != nil {
		fatalf("resolve root: %v", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		fatalf("resolve root symlinks: %v", err)
	}

	routes := loadRoutes(root, os.Args[2])
	var fixtures fixtureFile
	readJSON(os.Args[3], &fixtures)
	if fixtures.Version != 1 || fixtures.Kind != "go-ast-route-tests" {
		fatalf("unsupported fixture contract: version=%d kind=%q", fixtures.Version, fixtures.Kind)
	}
	if len(fixtures.Cases) == 0 {
		fatalf("fixture file contains no cases")
	}

	files := indexGo(root)
	seenNames := map[string]struct{}{}
	for _, tc := range fixtures.Cases {
		if strings.TrimSpace(tc.Name) == "" {
			fatalf("fixture name must not be empty")
		}
		if _, ok := seenNames[tc.Name]; ok {
			fatalf("duplicate fixture name %q", tc.Name)
		}
		seenNames[tc.Name] = struct{}{}

		r, ok := routes[tc.RouteID]
		if !ok {
			fatalf("%s: unknown route %q", tc.Name, tc.RouteID)
		}
		if r.SourceSelectors == nil {
			fatalf("%s: route %q has no source_selectors", tc.Name, tc.RouteID)
		}
		validateSelectors(tc.RouteID, *r.SourceSelectors)

		paths, reasons := selectSources(files, r)
		if !equalStrings(paths, tc.ExpectPaths) {
			fatalf("%s: paths got %v, want %v", tc.Name, paths, tc.ExpectPaths)
		}
		if len(tc.ExpectReasons) != len(paths) {
			fatalf("%s: expected reasons cover %d paths, selected %d", tc.Name, len(tc.ExpectReasons), len(paths))
		}
		for _, path := range paths {
			expected, ok := tc.ExpectReasons[path]
			if !ok {
				fatalf("%s: missing expected reasons for %s", tc.Name, path)
			}
			actual := append([]string(nil), reasons[path]...)
			sort.Strings(actual)
			want := append([]string(nil), expected...)
			sort.Strings(want)
			if !equalStrings(actual, want) {
				fatalf("%s: reasons for %s got %v, want %v", tc.Name, path, actual, want)
			}
		}
		fmt.Printf("ok: %s\n", tc.Name)
	}
	fmt.Printf("validated %d deterministic Go AST routing fixtures\n", len(fixtures.Cases))
}

func loadRoutes(root, routerPath string) map[string]route {
	var r router
	readYAML(routerPath, &r)
	if r.Version != 1 || r.Kind != "document-router" {
		fatalf("unsupported router contract: version=%d kind=%q", r.Version, r.Kind)
	}

	routes := map[string]route{}
	registerRoutes(routes, r.Routes, "root router")
	for _, configured := range r.Manifests {
		clean := filepath.Clean(configured)
		if configured == "" || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			fatalf("unsafe manifest path %q", configured)
		}
		resolved, err := filepath.EvalSymlinks(filepath.Join(root, clean))
		if err != nil {
			fatalf("resolve manifest %q: %v", configured, err)
		}
		if !withinRoot(root, resolved) {
			fatalf("manifest %q resolves outside repository", configured)
		}
		var m manifest
		readYAML(resolved, &m)
		if m.Version != 1 || m.Kind != "document-route-manifest" || strings.TrimSpace(m.Owner) == "" {
			fatalf("invalid manifest %q", configured)
		}
		registerRoutes(routes, m.Routes, configured)
	}
	return routes
}

func registerRoutes(target map[string]route, candidates []route, source string) {
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.ID) == "" {
			fatalf("route in %s has empty id", source)
		}
		if _, exists := target[candidate.ID]; exists {
			fatalf("duplicate route id %q", candidate.ID)
		}
		if candidate.SourceSelectors != nil {
			validateSelectors(candidate.ID, *candidate.SourceSelectors)
		}
		target[candidate.ID] = candidate
	}
}

func validateSelectors(routeID string, s sourceSelectors) {
	if s.Language != "go" {
		fatalf("route %q source_selectors language must be go", routeID)
	}
	if len(s.Packages)+len(s.Symbols)+len(s.Commands)+len(s.Files) == 0 {
		fatalf("route %q source_selectors must configure at least one selector family", routeID)
	}
	for _, value := range append(append([]string{}, s.Packages...), s.Commands...) {
		if !safeRelative(value) {
			fatalf("route %q has unsafe source selector %q", routeID, value)
		}
	}
	for _, symbol := range s.Symbols {
		if !token.IsIdentifier(symbol) || !ast.IsExported(symbol) {
			fatalf("route %q has invalid exported symbol selector %q", routeID, symbol)
		}
	}
	for _, pattern := range s.Files {
		if !safeRelative(pattern) {
			fatalf("route %q has unsafe file selector %q", routeID, pattern)
		}
		if _, err := filepath.Match(filepath.FromSlash(pattern), filepath.FromSlash(pattern)); err != nil {
			fatalf("route %q has invalid file selector %q: %v", routeID, pattern, err)
		}
	}
}

func safeRelative(value string) bool {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(value)))
	return value != "" && !filepath.IsAbs(clean) && clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator))
}

func withinRoot(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func indexGo(root string) []sourceFile {
	var out []sourceFile
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == "vendor" || name == "bin" || name == "dist" || name == "artifacts" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuse symlinked Go source %s", path)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		if !withinRoot(root, resolved) {
			return fmt.Errorf("Go source resolves outside repository: %s", path)
		}
		rel, err := filepath.Rel(root, resolved)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		parsed, err := parser.ParseFile(token.NewFileSet(), resolved, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", rel, err)
		}
		symbols := map[string]struct{}{}
		for _, decl := range parsed.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if ast.IsExported(d.Name.Name) {
					symbols[d.Name.Name] = struct{}{}
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if ast.IsExported(s.Name.Name) {
							symbols[s.Name.Name] = struct{}{}
						}
					case *ast.ValueSpec:
						for _, name := range s.Names {
							if ast.IsExported(name.Name) {
								symbols[name.Name] = struct{}{}
							}
						}
					}
				}
			}
		}
		dir := filepath.ToSlash(filepath.Dir(rel))
		command := ""
		parts := strings.Split(dir, "/")
		if parsed.Name.Name == "main" && len(parts) == 2 && parts[0] == "cmd" {
			command = parts[1]
		}
		out = append(out, sourceFile{Path: rel, PackageDir: dir, Command: command, Symbols: symbols})
		return nil
	})
	if err != nil {
		fatalf("index Go source: %v", err)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func selectSources(files []sourceFile, r route) ([]string, map[string][]string) {
	s := *r.SourceSelectors
	var paths []string
	reasons := map[string][]string{}
	for _, file := range files {
		if !allowedByRoute(file.Path, r.Include, r.Exclude) {
			continue
		}
		matched, ok := selectorReasons(file, s)
		if !ok {
			continue
		}
		sort.Strings(matched)
		paths = append(paths, file.Path)
		reasons[file.Path] = matched
	}
	sort.Strings(paths)
	return paths, reasons
}

func allowedByRoute(path string, include, exclude []string) bool {
	included := false
	for _, pattern := range include {
		if matchPath(pattern, path) {
			included = true
			break
		}
	}
	if !included {
		return false
	}
	for _, pattern := range exclude {
		if matchPath(pattern, path) {
			return false
		}
	}
	return true
}

func matchPath(pattern, path string) bool {
	pattern = filepath.ToSlash(pattern)
	path = filepath.ToSlash(path)
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
	matched, err := filepath.Match(filepath.FromSlash(pattern), filepath.FromSlash(path))
	if err != nil {
		fatalf("invalid path pattern %q: %v", pattern, err)
	}
	return matched
}

func selectorReasons(file sourceFile, s sourceSelectors) ([]string, bool) {
	var matched []string
	if len(s.Packages) > 0 {
		ok := false
		for _, pkg := range s.Packages {
			if file.PackageDir == strings.TrimSuffix(filepath.ToSlash(pkg), "/") {
				ok = true
				matched = append(matched, "package:"+pkg)
				break
			}
		}
		if !ok {
			return nil, false
		}
	}
	if len(s.Symbols) > 0 {
		ok := false
		for _, symbol := range s.Symbols {
			if _, exists := file.Symbols[symbol]; exists {
				ok = true
				matched = append(matched, "symbol:"+symbol)
			}
		}
		if !ok {
			return nil, false
		}
	}
	if len(s.Commands) > 0 {
		ok := false
		for _, command := range s.Commands {
			if file.Command == command {
				ok = true
				matched = append(matched, "command:"+command)
				break
			}
		}
		if !ok {
			return nil, false
		}
	}
	if len(s.Files) > 0 {
		ok := false
		for _, pattern := range s.Files {
			if matchPath(pattern, file.Path) {
				ok = true
				matched = append(matched, "file:"+pattern)
				break
			}
		}
		if !ok {
			return nil, false
		}
	}
	return matched, len(matched) > 0
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
	fmt.Fprintf(os.Stderr, "go-ast-routing: "+format+"\n", args...)
	os.Exit(1)
}
