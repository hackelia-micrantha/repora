package posture

import "testing"

func TestGitHubFileFactsRespectProviderPathCasing(t *testing.T) {
	tree := GitHubTree{Entries: []GitHubTreeEntry{
		{Path: ".github/codeowners", Type: "blob", SHA: "wrong-codeowners"},
		{Path: ".GITHUB/workflows/ci.yml", Type: "blob", SHA: "wrong-workflow-dir"},
		{Path: ".github/issue_template/bug.yml", Type: "blob", SHA: "wrong-issue-dir"},
		{Path: ".github/PULL_REQUEST_TEMPLATE.MD", Type: "blob", SHA: "pr"},
	}}
	inventory := NewInventory("acme/project")
	populateTreeFacts(&inventory, tree, Evidence{Source: "github.git_tree", Reference: "tree"})

	if inventory.RepositoryFacts.CODEOWNERSPresent.State != StateObserved || inventory.RepositoryFacts.CODEOWNERSPresent.Value == nil || *inventory.RepositoryFacts.CODEOWNERSPresent.Value {
		t.Fatalf("wrong-case CODEOWNERS should not be recognized: %#v", inventory.RepositoryFacts.CODEOWNERSPresent)
	}
	if inventory.RepositoryFacts.IssueTemplatePresent.State != StateObserved || inventory.RepositoryFacts.IssueTemplatePresent.Value == nil || *inventory.RepositoryFacts.IssueTemplatePresent.Value {
		t.Fatalf("wrong-case ISSUE_TEMPLATE directory should not be recognized: %#v", inventory.RepositoryFacts.IssueTemplatePresent)
	}
	if inventory.RepositoryFacts.PullRequestTemplatePresent.Value == nil || !*inventory.RepositoryFacts.PullRequestTemplatePresent.Value {
		t.Fatalf("pull-request template filename should be case-insensitive: %#v", inventory.RepositoryFacts.PullRequestTemplatePresent)
	}
	if inventory.RepositoryFacts.WorkflowPaths.Value == nil || len(*inventory.RepositoryFacts.WorkflowPaths.Value) != 0 {
		t.Fatalf("wrong-case Actions directory should not be recognized: %#v", inventory.RepositoryFacts.WorkflowPaths)
	}
}

func TestGitHubPullRequestTemplateSupportedLocations(t *testing.T) {
	for _, path := range []string{
		"pull_request_template.txt",
		"docs/PULL_REQUEST_TEMPLATE.md",
		"PULL_REQUEST_TEMPLATE/a.md",
		"docs/PULL_REQUEST_TEMPLATE/a.md",
		".github/PULL_REQUEST_TEMPLATE/a.md",
	} {
		t.Run(path, func(t *testing.T) {
			entries := map[string]GitHubTreeEntry{path: {Path: path, Type: "blob", SHA: "p"}}
			fact := pullRequestTemplatePresence(entries, true, Evidence{Source: "test", Reference: path})
			if fact.State != StateObserved || fact.Value == nil || !*fact.Value {
				t.Fatalf("supported pull-request template location not detected: %#v", fact)
			}
		})
	}
}
