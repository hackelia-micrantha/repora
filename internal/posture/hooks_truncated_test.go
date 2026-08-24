package posture

import (
	"context"
	"testing"
)

func TestCollectGitHubHooksKeepsAbsentManagerUnknownForTruncatedTree(t *testing.T) {
	reader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repo"),
		branch:        GitHubBranch{Name: "main", CommitSHA: "abc", TreeSHA: "tree"},
		branchObs:     available("github.branch", "main"),
		tree: GitHubTree{Truncated: true, Entries: []GitHubTreeEntry{
			{Path: hooksProfilePath, Type: "blob", SHA: "profile"},
		}},
		treeObs: available("github.git_tree", "tree"),
		blobs: map[string][]byte{
			"profile": []byte("kind: repora.posture-hooks-profile\nversion: 1\n"),
		},
	}

	inventory, err := CollectGitHubHooks(context.Background(), reader, "acme/project")
	if err != nil {
		t.Fatal(err)
	}
	if inventory.Manager.State != StateUnknown || inventory.Manager.Value != nil {
		t.Fatalf("manager = %#v", inventory.Manager)
	}
}

func TestCollectGitHubHooksKeepsNegativeCICoverageUnknownForTruncatedTree(t *testing.T) {
	reader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repo"),
		branch:        GitHubBranch{Name: "main", CommitSHA: "abc", TreeSHA: "tree"},
		branchObs:     available("github.branch", "main"),
		tree: GitHubTree{Truncated: true, Entries: []GitHubTreeEntry{
			{Path: hooksProfilePath, Type: "blob", SHA: "profile"},
			{Path: ".github/workflows/visible.yml", Type: "blob", SHA: "workflow"},
		}},
		treeObs: available("github.git_tree", "tree"),
		blobs: map[string][]byte{
			"profile":  []byte("kind: repora.posture-hooks-profile\nversion: 1\nrequired_checks: [gofmt]\n"),
			"workflow": []byte("name: visible\njobs: {}\n"),
		},
	}

	inventory, err := CollectGitHubHooks(context.Background(), reader, "acme/project")
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.RequiredChecks) != 1 || inventory.RequiredChecks[0].CICovered.State != StateUnknown {
		t.Fatalf("required checks = %#v", inventory.RequiredChecks)
	}
}

func TestCollectGitHubHooksCanObservePositiveCICoverageFromTruncatedTree(t *testing.T) {
	reader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repo"),
		branch:        GitHubBranch{Name: "main", CommitSHA: "abc", TreeSHA: "tree"},
		branchObs:     available("github.branch", "main"),
		tree: GitHubTree{Truncated: true, Entries: []GitHubTreeEntry{
			{Path: hooksProfilePath, Type: "blob", SHA: "profile"},
			{Path: ".github/workflows/visible.yml", Type: "blob", SHA: "workflow"},
		}},
		treeObs: available("github.git_tree", "tree"),
		blobs: map[string][]byte{
			"profile":  []byte("kind: repora.posture-hooks-profile\nversion: 1\nrequired_checks: [gofmt]\n"),
			"workflow": []byte("name: visible\njobs:\n  format:\n    steps:\n      - run: gofmt -w .\n"),
		},
	}

	inventory, err := CollectGitHubHooks(context.Background(), reader, "acme/project")
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.RequiredChecks) != 1 || !boolValue(t, inventory.RequiredChecks[0].CICovered) {
		t.Fatalf("required checks = %#v", inventory.RequiredChecks)
	}
}

func TestCollectGitHubHooksSkipsWorkflowReadsWithoutRequiredChecks(t *testing.T) {
	reader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repo"),
		branch:        GitHubBranch{Name: "main", CommitSHA: "abc", TreeSHA: "tree"},
		branchObs:     available("github.branch", "main"),
		tree: GitHubTree{Entries: []GitHubTreeEntry{
			{Path: ".github/workflows/ci.yml", Type: "blob", SHA: "unreadable-if-requested"},
		}},
		treeObs: available("github.git_tree", "tree"),
		blobs:   map[string][]byte{},
	}

	inventory, err := CollectGitHubHooks(context.Background(), reader, "acme/project")
	if err != nil {
		t.Fatalf("baseline collection unexpectedly read workflow: %v", err)
	}
	if len(inventory.RequiredChecks) != 0 {
		t.Fatalf("required checks = %#v", inventory.RequiredChecks)
	}
}
