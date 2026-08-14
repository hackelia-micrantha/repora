package posture

import "testing"

func TestIssueTemplatePresenceIgnoresChooserConfig(t *testing.T) {
	entries := map[string]GitHubTreeEntry{
		".github/ISSUE_TEMPLATE/config.yml": {Path: ".github/ISSUE_TEMPLATE/config.yml", Type: "blob", SHA: "config"},
	}
	fact := issueTemplatePresence(entries, true, Evidence{Source: "github.git_tree", Reference: "tree"})
	if fact.State != StateObserved || fact.Value == nil || *fact.Value {
		t.Fatalf("chooser config should not count as an issue template: %#v", fact)
	}
}

func TestIssueTemplatePresenceAcceptsMarkdownAndIssueForms(t *testing.T) {
	for _, path := range []string{
		".github/ISSUE_TEMPLATE/bug.md",
		".github/ISSUE_TEMPLATE/bug.yml",
		".github/ISSUE_TEMPLATE/bug.yaml",
	} {
		t.Run(path, func(t *testing.T) {
			entries := map[string]GitHubTreeEntry{
				path: {Path: path, Type: "blob", SHA: "template"},
			}
			fact := issueTemplatePresence(entries, true, Evidence{Source: "github.git_tree", Reference: "tree"})
			if fact.State != StateObserved || fact.Value == nil || !*fact.Value {
				t.Fatalf("issue template was not detected: %#v", fact)
			}
		})
	}
}
