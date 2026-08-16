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

func TestMarkdownRepositoryLinksIgnoreCodeExamples(t *testing.T) {
	links := markdownRepositoryLinks("README.md", []byte("See [architecture](docs/architecture.md).\n\n```markdown\n[security](SECURITY.md)\n```\n\n    [license](LICENSE)\n\n   [plan](docs/plans/current.md)\n"))
	for _, want := range []string{"docs/architecture.md", "docs/plans/current.md"} {
		if _, ok := links[want]; !ok {
			t.Fatalf("link %q missing from %#v", want, links)
		}
	}
	for _, unwanted := range []string{"SECURITY.md", "LICENSE"} {
		if _, ok := links[unwanted]; ok {
			t.Fatalf("code example link %q unexpectedly observed in %#v", unwanted, links)
		}
	}
}
