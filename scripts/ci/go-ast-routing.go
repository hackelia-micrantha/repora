//go:build ignore

// go-ast-routing validates deterministic Go source selectors against repository source.
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
)

type selectors struct {
	Packages []string `json:"packages"`
	Symbols  []string `json:"symbols"`
	Commands []string `json:"commands"`
	Files    []string `json:"files"`
}

type fixtureFile struct {
	Version int       `json:"version"`
	Kind    string    `json:"kind"`
	Cases   []fixture `json:"cases"`
}

type fixture struct {
	Name          string              `json:"name"`
	Selectors     selectors           `json:"selectors"`
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
	if len(os.Args) != 3 {
		fatalf("usage: go run ./scripts/ci/go-ast-routing.go <repo-root> <fixtures.json>")
	}
	root, err := filepath.Abs(os.Args[1])
	if err != nil {
		fatalf("resolve root: %v", err)
	}
	var fixtures fixtureFile
	readJSON(os.Args[2], &fixtures)
	if fixtures.Version != 1 || fixtures.Kind != "go-ast-route-tests" {
		fatalf("unsupported fixture contract: version=%d kind=%q", fixtures.Version, fixtures.Kind)
	}
	files := indexGo(root)
	for _, tc := range fixtures.Cases {
		paths, reasons := selectSources(files, tc.Selectors)
		if !equalStrings(paths, tc.ExpectPaths) {
			fatalf("%s: paths got %v, want %v", tc.Name, paths, tc.ExpectPaths)
		}
		for path, expected := range tc.ExpectReasons {
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
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
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
		if len(parts) == 2 && parts[0] == "cmd" {
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

func selectSources(files []sourceFile, s selectors) ([]string, map[string][]string) {
	var paths []string
	reasons := map[string][]string{}
	for _, file := range files {
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
				continue
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
				continue
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
				continue
			}
		}
		if len(s.Files) > 0 {
			ok := false
			for _, pattern := range s.Files {
				match, err := filepath.Match(filepath.FromSlash(pattern), filepath.FromSlash(file.Path))
				if err != nil {
					fatalf("invalid file selector %q: %v", pattern, err)
				}
				if match {
					ok = true
					matched = append(matched, "file:"+pattern)
					break
				}
			}
			if !ok {
				continue
			}
		}
		if len(matched) == 0 {
			continue
		}
		sort.Strings(matched)
		paths = append(paths, file.Path)
		reasons[file.Path] = matched
	}
	sort.Strings(paths)
	return paths, reasons
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
