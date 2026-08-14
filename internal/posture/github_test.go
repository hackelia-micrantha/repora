package posture

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeGitHubReader struct {
	repository    GitHubRepository
	repositoryObs ReadObservation
	branch        GitHubBranch
	branchObs     ReadObservation
	protection    GitHubProtection
	protectionObs ReadObservation
	tree          GitHubTree
	treeObs       ReadObservation
	blobs         map[string][]byte
	blobObs       map[string]ReadObservation
}

func (f fakeGitHubReader) Repository(context.Context, string) (GitHubRepository, ReadObservation, error) {
	return f.repository, f.repositoryObs, nil
}
func (f fakeGitHubReader) Branch(context.Context, string, string) (GitHubBranch, ReadObservation, error) {
	return f.branch, f.branchObs, nil
}
func (f fakeGitHubReader) BranchProtection(context.Context, string, string) (GitHubProtection, ReadObservation, error) {
	return f.protection, f.protectionObs, nil
}
func (f fakeGitHubReader) Tree(context.Context, string, string) (GitHubTree, ReadObservation, error) {
	return f.tree, f.treeObs, nil
}
func (f fakeGitHubReader) Blob(_ context.Context, _ string, sha string) ([]byte, ReadObservation, error) {
	data, ok := f.blobs[sha]
	if !ok {
		return nil, ReadObservation{}, fmt.Errorf("unexpected blob %s", sha)
	}
	obs, ok := f.blobObs[sha]
	if !ok {
		obs = ReadObservation{Available: true, Evidence: Evidence{Source: "github.blob", Reference: sha}}
	}
	return data, obs, nil
}

func available(source, reference string) ReadObservation {
	return ReadObservation{Available: true, Evidence: Evidence{Source: source, Reference: reference}}
}

func unavailable(source, reference string) ReadObservation {
	return ReadObservation{Available: false, Evidence: Evidence{Source: source, Reference: reference, Detail: "HTTP 403; evidence unavailable under current access"}}
}

func boolValue(t *testing.T, fact Fact[bool]) bool {
	t.Helper()
	if fact.State != StateObserved || fact.Value == nil {
		t.Fatalf("fact is not observed: %#v", fact)
	}
	return *fact.Value
}

func TestCollectGitHubBuildsRepositoryAndWorkflowFacts(t *testing.T) {
	allow := false
	reader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repos/acme/project"),
		branch:        GitHubBranch{Name: "main", Protected: true, CommitSHA: "abc1234", TreeSHA: "tree123"},
		branchObs:     available("github.branch", "repos/acme/project/branches/main"),
		protection: GitHubProtection{
			RequiredStatusChecks: []string{"ci", "security", "ci"},
			RequiredReviews:      2,
			AllowForcePushes:     &allow,
			AllowDeletions:       &allow,
		},
		protectionObs: available("github.branch_protection", "repos/acme/project/branches/main/protection"),
		tree: GitHubTree{Entries: []GitHubTreeEntry{
			{Path: ".github/CODEOWNERS", Type: "blob", SHA: "c"},
			{Path: ".github/ISSUE_TEMPLATE/bug.yml", Type: "blob", SHA: "i"},
			{Path: ".github/PULL_REQUEST_TEMPLATE.md", Type: "blob", SHA: "p"},
			{Path: ".github/SECURITY.md", Type: "blob", SHA: "s"},
			{Path: ".github/dependabot.yml", Type: "blob", SHA: "d"},
			{Path: ".github/workflows/ci.yml", Type: "blob", SHA: "w"},
			{Path: "LICENSE", Type: "blob", SHA: "l"},
		}},
		treeObs: available("github.git_tree", "acme/project:tree123"),
		blobs: map[string][]byte{
			"w": []byte(`on:
  pull_request_target:
permissions:
  contents: read
jobs:
  test:
    runs-on: [self-hosted, linux]
    steps:
      - uses: vendor/action@0123456789012345678901234567890123456789
`),
		},
	}

	inventory, err := CollectGitHub(context.Background(), reader, "acme/project")
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if err := inventory.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if inventory.RepositoryFacts.DefaultBranch.Value == nil || *inventory.RepositoryFacts.DefaultBranch.Value != "main" {
		t.Fatalf("default branch = %#v", inventory.RepositoryFacts.DefaultBranch)
	}
	if !boolValue(t, inventory.RepositoryFacts.DefaultBranchProtected) || !boolValue(t, inventory.RepositoryFacts.ForcePushProtected) || !boolValue(t, inventory.RepositoryFacts.DeletionProtected) {
		t.Fatal("branch protection facts were not normalized")
	}
	if inventory.RepositoryFacts.RequiredReviews.Value == nil || *inventory.RepositoryFacts.RequiredReviews.Value != 2 {
		t.Fatalf("required reviews = %#v", inventory.RepositoryFacts.RequiredReviews)
	}
	checks := *inventory.RepositoryFacts.RequiredStatusChecks.Value
	if len(checks) != 2 || checks[0] != "ci" || checks[1] != "security" {
		t.Fatalf("required checks = %#v", checks)
	}
	for name, fact := range map[string]Fact[bool]{
		"codeowners": inventory.RepositoryFacts.CODEOWNERSPresent,
		"security":   inventory.RepositoryFacts.SecurityMDPresent,
		"license":    inventory.RepositoryFacts.LicensePresent,
		"issue":      inventory.RepositoryFacts.IssueTemplatePresent,
		"pr":         inventory.RepositoryFacts.PullRequestTemplatePresent,
	} {
		if !boolValue(t, fact) {
			t.Fatalf("%s presence = false", name)
		}
	}
	if inventory.RepositoryFacts.DependencyAutomation.Value == nil || len(*inventory.RepositoryFacts.DependencyAutomation.Value) != 1 || (*inventory.RepositoryFacts.DependencyAutomation.Value)[0] != "dependabot" {
		t.Fatalf("dependency automation = %#v", inventory.RepositoryFacts.DependencyAutomation)
	}
	if inventory.WorkflowsState != StateObserved || len(inventory.Workflows) != 1 {
		t.Fatalf("workflows = state %q %#v", inventory.WorkflowsState, inventory.Workflows)
	}
	workflow := inventory.Workflows[0]
	if !workflow.UsesPullRequestTarget || len(workflow.Jobs) != 1 || !boolValue(t, workflow.Jobs[0].SelfHosted) {
		t.Fatalf("workflow facts = %#v", workflow)
	}
	if len(workflow.Jobs[0].Actions) != 1 || !workflow.Jobs[0].Actions[0].ThirdParty || workflow.Jobs[0].Actions[0].Pinning != "immutable-sha" {
		t.Fatalf("action facts = %#v", workflow.Jobs[0].Actions)
	}
}

func TestCollectGitHubPreservesProviderUnavailability(t *testing.T) {
	reader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repos/acme/project"),
		branch:        GitHubBranch{Name: "main", Protected: true, CommitSHA: "abc1234", TreeSHA: "tree123"},
		branchObs:     available("github.branch", "repos/acme/project/branches/main"),
		protectionObs: unavailable("github.branch_protection", "repos/acme/project/branches/main/protection"),
		tree:          GitHubTree{Entries: []GitHubTreeEntry{{Path: "LICENSE", Type: "blob", SHA: "l"}}},
		treeObs:       available("github.git_tree", "acme/project:tree123"),
		blobs:         map[string][]byte{},
	}
	inventory, err := CollectGitHub(context.Background(), reader, "acme/project")
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if inventory.RepositoryFacts.RequiredReviews.State != StateUnavailable || inventory.RepositoryFacts.ForcePushProtected.State != StateUnavailable {
		t.Fatalf("provider-protected facts = %#v %#v", inventory.RepositoryFacts.RequiredReviews, inventory.RepositoryFacts.ForcePushProtected)
	}
	if !boolValue(t, inventory.RepositoryFacts.LicensePresent) {
		t.Fatal("file-backed fact should remain available when protection API is unavailable")
	}
	if boolValue(t, inventory.RepositoryFacts.SecurityMDPresent) {
		t.Fatal("complete tree should report missing SECURITY.md as observed false")
	}
}

func TestCollectGitHubUsesUnknownForNegativeFactsFromTruncatedTree(t *testing.T) {
	reader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repos/acme/project"),
		branch:        GitHubBranch{Name: "main", Protected: false, CommitSHA: "abc1234", TreeSHA: "tree123"},
		branchObs:     available("github.branch", "repos/acme/project/branches/main"),
		tree: GitHubTree{Truncated: true, Entries: []GitHubTreeEntry{
			{Path: ".github/CODEOWNERS", Type: "blob", SHA: "c"},
		}},
		treeObs: available("github.git_tree", "acme/project:tree123"),
		blobs:   map[string][]byte{},
	}
	inventory, err := CollectGitHub(context.Background(), reader, "acme/project")
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if !boolValue(t, inventory.RepositoryFacts.CODEOWNERSPresent) {
		t.Fatal("present fact should remain observed in truncated tree")
	}
	if inventory.RepositoryFacts.SecurityMDPresent.State != StateUnknown || inventory.RepositoryFacts.SecurityMDPresent.Value != nil {
		t.Fatalf("missing fact from truncated tree = %#v", inventory.RepositoryFacts.SecurityMDPresent)
	}
	if inventory.WorkflowsState != StateUnknown || inventory.RepositoryFacts.WorkflowPaths.State != StateUnknown {
		t.Fatalf("workflow completeness = %q %#v", inventory.WorkflowsState, inventory.RepositoryFacts.WorkflowPaths)
	}
}

func TestCollectGitHubReturnsUnavailableInventoryWhenRepositoryCannotBeRead(t *testing.T) {
	reader := fakeGitHubReader{repositoryObs: unavailable("github.repository", "repos/acme/private")}
	inventory, err := CollectGitHub(context.Background(), reader, "acme/private")
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if inventory.RepositoryFacts.DefaultBranch.State != StateUnavailable || inventory.RepositoryFacts.LicensePresent.State != StateUnavailable || inventory.WorkflowsState != StateUnavailable {
		t.Fatalf("unavailable inventory = %#v", inventory)
	}
}

func TestHTTPGitHubReaderUsesGETAndDoesNotExposeTokenInEvidence(t *testing.T) {
	const token = "super-secret-token"
	var method, auth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		auth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"default_branch":"main"}`)
	}))
	defer server.Close()

	reader := NewHTTPGitHubReader(token)
	reader.BaseURL = server.URL
	repository, obs, err := reader.Repository(context.Background(), "acme/project")
	if err != nil {
		t.Fatalf("repository: %v", err)
	}
	if repository.DefaultBranch != "main" || method != http.MethodGet || auth != "Bearer "+token {
		t.Fatalf("request method/auth/repository = %q %q %#v", method, auth, repository)
	}
	if strings.Contains(obs.Evidence.Reference, token) || strings.Contains(obs.Evidence.Detail, token) {
		t.Fatal("token leaked into posture evidence")
	}
}
