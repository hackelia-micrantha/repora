package posture

import "testing"

func TestMarkdownHeadingsPreserveHashesAndIgnoreIndentedCode(t *testing.T) {
	headings := markdownHeadings([]byte("## C#\n\n   ### Allowed\n\n    ## Indented code\n\n    ```\n## After indented code fence\n    ```\n\n## Security ##\n\n```markdown\n## Fenced example\n```\n"))
	for _, want := range []string{"c#", "allowed", "after indented code fence", "security"} {
		if _, ok := headings[want]; !ok {
			t.Fatalf("heading %q missing from %#v", want, headings)
		}
	}
	for _, unwanted := range []string{"c", "indented code", "fenced example"} {
		if _, ok := headings[unwanted]; ok {
			t.Fatalf("unexpected heading %q in %#v", unwanted, headings)
		}
	}
}

func TestNormalizeHeadingOnlyStripsWhitespaceSeparatedClosingHashes(t *testing.T) {
	for input, want := range map[string]string{
		"C#":            "c#",
		"Security ##":   "security",
		"API ### notes": "api ### notes",
	} {
		if got := normalizeHeading(input); got != want {
			t.Fatalf("normalizeHeading(%q) = %q, want %q", input, got, want)
		}
	}
}
