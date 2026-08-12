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
	want, err := os.ReadFile(filepath.Join("testdata", "readme.golden.md"))
	if err != nil {
		t.Fatal(err)
	}

	got, err := RenderREADME(template, RenderData{
		RepoID:            "repora",
		RepoUID:           "repo.repora",
		CanonicalProvider: "gitlab",
		CanonicalPath:     "micrantha/repora",
		Values: map[string]string{
			"title":   "Repora",
			"summary": "Deterministic repository mirror management",
			"literal": "{{repo.uid}}",
		},
	})
	if err != nil {
		t.Fatalf("RenderREADME() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("RenderREADME() = %q, want %q", got, want)
	}
}

func TestRenderREADMENormalizesLineEndings(t *testing.T) {
	got, err := RenderREADME([]byte("# {{value.title}}\r\n{{value.body}}\r"), RenderData{
		Values: map[string]string{
			"title": "Title\r\nLine",
			"body":  "Body\rText",
		},
	})
	if err != nil {
		t.Fatalf("RenderREADME() error = %v", err)
	}
	if want := "# Title\nLine\nBody\nText\n"; string(got) != want {
		t.Fatalf("RenderREADME() = %q, want %q", got, want)
	}
}

func TestRenderREADMERejectsUnknownToken(t *testing.T) {
	_, err := RenderREADME([]byte("{{unknown}}"), RenderData{})
	if err == nil || !strings.Contains(err.Error(), "unsupported or unresolved token") {
		t.Fatalf("RenderREADME() error = %v, want unknown-token rejection", err)
	}
}

func TestRenderREADMERejectsMissingConfiguredValue(t *testing.T) {
	_, err := RenderREADME([]byte("{{value.title}}"), RenderData{})
	if err == nil || !strings.Contains(err.Error(), `"value.title"`) {
		t.Fatalf("RenderREADME() error = %v, want unresolved value token", err)
	}
}

func TestRenderREADMERejectsMissingBuiltin(t *testing.T) {
	_, err := RenderREADME([]byte("{{repo.id}}"), RenderData{})
	if err == nil || !strings.Contains(err.Error(), `"repo.id"`) {
		t.Fatalf("RenderREADME() error = %v, want unresolved built-in token", err)
	}
}

func TestRenderREADMEAllowsExplicitEmptyConfiguredValue(t *testing.T) {
	got, err := RenderREADME([]byte("before{{value.empty}}after"), RenderData{Values: map[string]string{"empty": ""}})
	if err != nil {
		t.Fatalf("RenderREADME() error = %v", err)
	}
	if want := "beforeafter"; string(got) != want {
		t.Fatalf("RenderREADME() = %q, want %q", got, want)
	}
}

func TestRenderREADMERejectsMalformedTokens(t *testing.T) {
	for _, template := range []string{"{{repo.id", "repo.id}}", "{{ repo.id }}", "{{repo{{id}}"} {
		_, err := RenderREADME([]byte(template), RenderData{RepoID: "repora"})
		if err == nil {
			t.Fatalf("RenderREADME(%q) returned nil error", template)
		}
	}
}

func TestRenderREADMERejectsNULAndInvalidUTF8(t *testing.T) {
	if _, err := RenderREADME([]byte{'a', 0, 'b'}, RenderData{}); err == nil {
		t.Fatal("RenderREADME accepted NUL template")
	}
	if _, err := RenderREADME([]byte{0xff}, RenderData{}); err == nil {
		t.Fatal("RenderREADME accepted invalid UTF-8 template")
	}
	if _, err := RenderREADME([]byte("{{value.x}}"), RenderData{Values: map[string]string{"x": "a\x00b"}}); err == nil {
		t.Fatal("RenderREADME accepted NUL replacement")
	}
}

func TestRenderREADMERejectsInvalidValueKeyDeterministically(t *testing.T) {
	_, err := RenderREADME([]byte("static"), RenderData{Values: map[string]string{
		"z bad": "value",
		"A bad": "value",
	}})
	if err == nil || !strings.Contains(err.Error(), `"A bad"`) {
		t.Fatalf("RenderREADME() error = %v, want lexicographically first invalid key", err)
	}
}

func TestRenderREADMERejectsOversizedTemplateAndOutput(t *testing.T) {
	oversized := bytes.Repeat([]byte{'a'}, MaxTextBytes+1)
	if _, err := RenderREADME(oversized, RenderData{}); err == nil {
		t.Fatal("RenderREADME accepted oversized template")
	}

	value := strings.Repeat("x", MaxTextBytes)
	_, err := RenderREADME([]byte("prefix{{value.large}}"), RenderData{Values: map[string]string{"large": value}})
	if err == nil || !strings.Contains(err.Error(), "rendered README exceeds") {
		t.Fatalf("RenderREADME() error = %v, want oversized output rejection", err)
	}
}
