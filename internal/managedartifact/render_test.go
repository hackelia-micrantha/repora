package managedartifact

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderREADMEGolden(t *testing.T) {
	template, err := os.ReadFile(filepath.Join("testdata", "readme.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	golden, err := os.ReadFile(filepath.Join("testdata", "readme.golden"))
	if err != nil {
		t.Fatal(err)
	}

	got, err := RenderREADME(template, RenderContext{
		RepoID:            "repora",
		RepoUID:           "repo.repora",
		CanonicalProvider: "gitlab",
		CanonicalPath:     "micrantha/repora",
		Values: map[string]string{
			"title":   "Repora",
			"summary": "Deterministic repository mirror management",
		},
	})
	if err != nil {
		t.Fatalf("RenderREADME error = %v", err)
	}
	if !bytes.Equal(got, golden) {
		t.Fatalf("rendered README mismatch\ngot:\n%s\nwant:\n%s", got, golden)
	}
}

func TestRenderREADMEDoesNotRecursivelyEvaluateValues(t *testing.T) {
	got, err := RenderREADME([]byte("{{value.title}}\n"), RenderContext{
		RepoID:            "repora",
		RepoUID:           "repo.repora",
		CanonicalProvider: "gitlab",
		CanonicalPath:     "micrantha/repora",
		Values:            map[string]string{"title": "{{repo.id}}"},
	})
	if err != nil {
		t.Fatalf("RenderREADME error = %v", err)
	}
	if string(got) != "{{repo.id}}\n" {
		t.Fatalf("got %q, want literal replacement token", got)
	}
}

func TestRenderREADMENormalizesLineEndings(t *testing.T) {
	got, err := RenderREADME([]byte("a\r\n{{value.text}}\rb\r"), RenderContext{
		RepoID:            "repora",
		RepoUID:           "repo.repora",
		CanonicalProvider: "gitlab",
		CanonicalPath:     "micrantha/repora",
		Values:            map[string]string{"text": "x\r\ny"},
	})
	if err != nil {
		t.Fatalf("RenderREADME error = %v", err)
	}
	if string(got) != "a\nx\ny\nb\n" {
		t.Fatalf("got %q, want normalized LF output", got)
	}
}

func TestRenderREADMERejectsUnknownMissingAndMalformedPlaceholders(t *testing.T) {
	context := RenderContext{
		RepoID:            "repora",
		RepoUID:           "repo.repora",
		CanonicalProvider: "gitlab",
		CanonicalPath:     "micrantha/repora",
		Values:            map[string]string{"title": "Repora"},
	}
	for name, template := range map[string]string{
		"unknown":   "{{repo.name}}",
		"missing":   "{{value.summary}}",
		"unclosed":  "{{value.title",
		"close-only": "value.title}}",
		"whitespace": "{{ value.title }}",
		"nested":    "{{value.{{title}}}}",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := RenderREADME([]byte(template), context); err == nil {
				t.Fatalf("RenderREADME accepted invalid template %q", template)
			}
		})
	}
}

func TestRenderREADMERejectsInvalidText(t *testing.T) {
	context := RenderContext{
		RepoID:            "repora",
		RepoUID:           "repo.repora",
		CanonicalProvider: "gitlab",
		CanonicalPath:     "micrantha/repora",
	}
	if _, err := RenderREADME([]byte{0xff}, context); err == nil {
		t.Fatal("RenderREADME accepted invalid UTF-8 template")
	}
	if _, err := RenderREADME([]byte{'a', 0, 'b'}, context); err == nil {
		t.Fatal("RenderREADME accepted NUL template")
	}
	context.Values = map[string]string{"title": "bad\x00value"}
	if _, err := RenderREADME([]byte("{{value.title}}"), context); err == nil {
		t.Fatal("RenderREADME accepted NUL replacement")
	}
}

func TestRenderREADMEEnforcesTextBounds(t *testing.T) {
	context := RenderContext{
		RepoID:            "repora",
		RepoUID:           "repo.repora",
		CanonicalProvider: "gitlab",
		CanonicalPath:     "micrantha/repora",
	}
	if _, err := RenderREADME(bytes.Repeat([]byte{'x'}, MaxTextBytes+1), context); err == nil {
		t.Fatal("RenderREADME accepted oversized template")
	}
	context.Values = map[string]string{"text": strings.Repeat("x", MaxTextBytes)}
	if _, err := RenderREADME([]byte("prefix{{value.text}}"), context); err == nil {
		t.Fatal("RenderREADME accepted oversized output")
	}
}
