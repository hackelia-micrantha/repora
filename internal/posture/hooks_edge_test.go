package posture

import (
	"context"
	"testing"
)

func TestCollectGitHubHooksReportsDeclaredMissingHook(t *testing.T) {
	reader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repo"),
		branch:        GitHubBranch{Name: "main", CommitSHA: "abc", TreeSHA: "tree"},
		branchObs:     available("github.branch", "main"),
		tree: GitHubTree{Entries: []GitHubTreeEntry{
			{Path: hooksProfilePath, Type: "blob", SHA: "profile"},
		}},
		treeObs: available("github.git_tree", "tree"),
		blobs: map[string][]byte{
			"profile": []byte("kind: repora.posture-hooks-profile\nversion: 1\nmanager: custom\nhook_paths: [.githooks/pre-commit]\n"),
		},
	}

	inventory, err := CollectGitHubHooks(context.Background(), reader, "acme/project")
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Entrypoints) != 1 {
		t.Fatalf("entrypoints = %#v", inventory.Entrypoints)
	}
	entrypoint := inventory.Entrypoints[0]
	if boolValue(t, entrypoint.Configured) {
		t.Fatal("declared missing hook reported configured")
	}
	if entrypoint.NetworkLoaded.State != StateUnknown || entrypoint.Executable.State != StateUnknown {
		t.Fatalf("missing hook trust facts = %#v %#v", entrypoint.NetworkLoaded, entrypoint.Executable)
	}
}

func TestCollectGitHubHooksMalformedProfileIsUnknown(t *testing.T) {
	reader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repo"),
		branch:        GitHubBranch{Name: "main", CommitSHA: "abc", TreeSHA: "tree"},
		branchObs:     available("github.branch", "main"),
		tree: GitHubTree{Entries: []GitHubTreeEntry{
			{Path: hooksProfilePath, Type: "blob", SHA: "profile"},
		}},
		treeObs: available("github.git_tree", "tree"),
		blobs: map[string][]byte{
			"profile": []byte("kind: repora.posture-hooks-profile\nversion: nope\n"),
		},
	}

	inventory, err := CollectGitHubHooks(context.Background(), reader, "acme/project")
	if err != nil {
		t.Fatal(err)
	}
	if inventory.ProfileDeclared.State != StateObserved || !boolValue(t, inventory.ProfileDeclared) {
		t.Fatalf("profile presence = %#v", inventory.ProfileDeclared)
	}
	if inventory.Manager.State != StateUnknown || inventory.BootstrapPresent.State != StateUnknown || inventory.BypassPresent.State != StateUnknown {
		t.Fatalf("malformed profile facts = %#v %#v %#v", inventory.Manager, inventory.BootstrapPresent, inventory.BypassPresent)
	}
}
