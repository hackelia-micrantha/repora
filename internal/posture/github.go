package posture

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const maxGitHubResponseBytes = 16 << 20

type ReadObservation struct {
	Available bool
	Evidence  Evidence
}

type GitHubRepository struct {
	DefaultBranch string
}

type GitHubBranch struct {
	Name      string
	Protected bool
	CommitSHA string
	TreeSHA   string
}

type GitHubProtection struct {
	RequiredStatusChecks []string
	RequiredReviews      int
	AllowForcePushes     *bool
	AllowDeletions       *bool
}

type GitHubTreeEntry struct {
	Path string
	Type string
	SHA  string
}

type GitHubTree struct {
	Entries   []GitHubTreeEntry
	Truncated bool
}

// GitHubReader is deliberately read-only. Posture inventory cannot reach a
// provider mutation through this capability boundary.
type GitHubReader interface {
	Repository(context.Context, string) (GitHubRepository, ReadObservation, error)
	Branch(context.Context, string, string) (GitHubBranch, ReadObservation, error)
	BranchProtection(context.Context, string, string) (GitHubProtection, ReadObservation, error)
	Tree(context.Context, string, string) (GitHubTree, ReadObservation, error)
	Blob(context.Context, string, string) ([]byte, ReadObservation, error)
}

type HTTPGitHubReader struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

func NewHTTPGitHubReader(token string) *HTTPGitHubReader {
	return &HTTPGitHubReader{
		BaseURL: "https://api.github.com",
		Token:   token,
		Client:  &http.Client{Timeout: 20 * time.Second},
	}
}

func (r *HTTPGitHubReader) Repository(ctx context.Context, fullName string) (GitHubRepository, ReadObservation, error) {
	owner, repo, err := splitGitHubFullName(fullName)
	if err != nil {
		return GitHubRepository{}, ReadObservation{}, err
	}
	path := fmt.Sprintf("/repos/%s/%s", url.PathEscape(owner), url.PathEscape(repo))
	var response struct {
		DefaultBranch string `json:"default_branch"`
	}
	obs, err := r.getJSON(ctx, path, "github.repository", path, &response)
	if err != nil || !obs.Available {
		return GitHubRepository{}, obs, err
	}
	if response.DefaultBranch == "" {
		return GitHubRepository{}, obs, fmt.Errorf("GitHub repository %s returned an empty default branch", fullName)
	}
	return GitHubRepository{DefaultBranch: response.DefaultBranch}, obs, nil
}

func (r *HTTPGitHubReader) Branch(ctx context.Context, fullName, branch string) (GitHubBranch, ReadObservation, error) {
	owner, repo, err := splitGitHubFullName(fullName)
	if err != nil {
		return GitHubBranch{}, ReadObservation{}, err
	}
	path := fmt.Sprintf("/repos/%s/%s/branches/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(branch))
	var response struct {
		Name      string `json:"name"`
		Protected bool   `json:"protected"`
		Commit    struct {
			SHA    string `json:"sha"`
			Commit struct {
				Tree struct {
					SHA string `json:"sha"`
				} `json:"tree"`
			} `json:"commit"`
		} `json:"commit"`
	}
	obs, err := r.getJSON(ctx, path, "github.branch", path, &response)
	if err != nil || !obs.Available {
		return GitHubBranch{}, obs, err
	}
	if response.Name == "" || response.Commit.SHA == "" || response.Commit.Commit.Tree.SHA == "" {
		return GitHubBranch{}, obs, fmt.Errorf("GitHub branch %s/%s returned incomplete commit/tree identity", fullName, branch)
	}
	return GitHubBranch{
		Name:      response.Name,
		Protected: response.Protected,
		CommitSHA: response.Commit.SHA,
		TreeSHA:   response.Commit.Commit.Tree.SHA,
	}, obs, nil
}

func (r *HTTPGitHubReader) BranchProtection(ctx context.Context, fullName, branch string) (GitHubProtection, ReadObservation, error) {
	owner, repo, err := splitGitHubFullName(fullName)
	if err != nil {
		return GitHubProtection{}, ReadObservation{}, err
	}
	path := fmt.Sprintf("/repos/%s/%s/branches/%s/protection", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(branch))
	var response struct {
		RequiredStatusChecks *struct {
			Contexts []string `json:"contexts"`
			Checks   []struct {
				Context string `json:"context"`
			} `json:"checks"`
		} `json:"required_status_checks"`
		RequiredPullRequestReviews *struct {
			RequiredApprovingReviewCount int `json:"required_approving_review_count"`
		} `json:"required_pull_request_reviews"`
		AllowForcePushes *struct {
			Enabled bool `json:"enabled"`
		} `json:"allow_force_pushes"`
		AllowDeletions *struct {
			Enabled bool `json:"enabled"`
		} `json:"allow_deletions"`
	}
	obs, err := r.getJSON(ctx, path, "github.branch_protection", path, &response)
	if err != nil || !obs.Available {
		return GitHubProtection{}, obs, err
	}
	checks := []string{}
	if response.RequiredStatusChecks != nil {
		checks = append(checks, response.RequiredStatusChecks.Contexts...)
		for _, check := range response.RequiredStatusChecks.Checks {
			checks = append(checks, check.Context)
		}
	}
	protection := GitHubProtection{RequiredStatusChecks: sortedUnique(checks)}
	if response.RequiredPullRequestReviews != nil {
		protection.RequiredReviews = response.RequiredPullRequestReviews.RequiredApprovingReviewCount
	}
	if response.AllowForcePushes != nil {
		value := response.AllowForcePushes.Enabled
		protection.AllowForcePushes = &value
	}
	if response.AllowDeletions != nil {
		value := response.AllowDeletions.Enabled
		protection.AllowDeletions = &value
	}
	return protection, obs, nil
}

func (r *HTTPGitHubReader) Tree(ctx context.Context, fullName, treeSHA string) (GitHubTree, ReadObservation, error) {
	owner, repo, err := splitGitHubFullName(fullName)
	if err != nil {
		return GitHubTree{}, ReadObservation{}, err
	}
	path := fmt.Sprintf("/repos/%s/%s/git/trees/%s?recursive=1", url.PathEscape(owner), url.PathEscape(repo), url.QueryEscape(treeSHA))
	var response struct {
		Truncated bool `json:"truncated"`
		Tree      []struct {
			Path string `json:"path"`
			Type string `json:"type"`
			SHA  string `json:"sha"`
		} `json:"tree"`
	}
	obs, err := r.getJSON(ctx, path, "github.git_tree", fmt.Sprintf("%s:%s", fullName, treeSHA), &response)
	if err != nil || !obs.Available {
		return GitHubTree{}, obs, err
	}
	entries := make([]GitHubTreeEntry, 0, len(response.Tree))
	for _, entry := range response.Tree {
		entries = append(entries, GitHubTreeEntry{Path: entry.Path, Type: entry.Type, SHA: entry.SHA})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return GitHubTree{Entries: entries, Truncated: response.Truncated}, obs, nil
}

func (r *HTTPGitHubReader) Blob(ctx context.Context, fullName, blobSHA string) ([]byte, ReadObservation, error) {
	owner, repo, err := splitGitHubFullName(fullName)
	if err != nil {
		return nil, ReadObservation{}, err
	}
	path := fmt.Sprintf("/repos/%s/%s/git/blobs/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(blobSHA))
	var response struct {
		Encoding string `json:"encoding"`
		Content  string `json:"content"`
	}
	obs, err := r.getJSON(ctx, path, "github.blob", fmt.Sprintf("%s:%s", fullName, blobSHA), &response)
	if err != nil || !obs.Available {
		return nil, obs, err
	}
	if response.Encoding != "base64" {
		return nil, obs, fmt.Errorf("GitHub blob %s uses unsupported encoding %q", blobSHA, response.Encoding)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(response.Content, "\n", ""))
	if err != nil {
		return nil, obs, fmt.Errorf("decode GitHub blob %s: %w", blobSHA, err)
	}
	return decoded, obs, nil
}

func (r *HTTPGitHubReader) getJSON(ctx context.Context, path, source, reference string, out any) (ReadObservation, error) {
	base := strings.TrimRight(r.BaseURL, "/")
	if base == "" {
		return ReadObservation{}, fmt.Errorf("GitHub API base URL is empty")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return ReadObservation{}, fmt.Errorf("create GitHub request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "repora-posture-inventory")
	if r.Token != "" {
		req.Header.Set("Authorization", "Bearer "+r.Token)
	}
	client := r.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return ReadObservation{}, fmt.Errorf("read GitHub API %s: %w", reference, err)
	}
	defer resp.Body.Close()
	evidence := Evidence{Source: source, Reference: reference}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		evidence.Detail = fmt.Sprintf("HTTP %d; evidence unavailable under current access", resp.StatusCode)
		return ReadObservation{Available: false, Evidence: evidence}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ReadObservation{}, fmt.Errorf("read GitHub API %s: HTTP %d", reference, resp.StatusCode)
	}
	reader := io.LimitReader(resp.Body, maxGitHubResponseBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return ReadObservation{}, fmt.Errorf("read GitHub API response %s: %w", reference, err)
	}
	if len(data) > maxGitHubResponseBytes {
		return ReadObservation{}, fmt.Errorf("GitHub API response %s exceeds %d bytes", reference, maxGitHubResponseBytes)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return ReadObservation{}, fmt.Errorf("decode GitHub API response %s: %w", reference, err)
	}
	return ReadObservation{Available: true, Evidence: evidence}, nil
}

func CollectGitHub(ctx context.Context, reader GitHubReader, fullName string) (Inventory, error) {
	if _, _, err := splitGitHubFullName(fullName); err != nil {
		return Inventory{}, err
	}
	inventory := NewInventory(fullName)
	repository, repoObs, err := reader.Repository(ctx, fullName)
	if err != nil {
		return Inventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, repoObs.Evidence)
	if !repoObs.Available {
		setRepositoryUnavailable(&inventory, repoObs.Evidence)
		inventory.WorkflowsState = StateUnavailable
		return inventory, inventory.Validate()
	}

	inventory.RepositoryFacts.DefaultBranch = Observed(repository.DefaultBranch, repoObs.Evidence)
	branch, branchObs, err := reader.Branch(ctx, fullName, repository.DefaultBranch)
	if err != nil {
		return Inventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, branchObs.Evidence)
	if !branchObs.Available {
		setBranchUnavailable(&inventory, branchObs.Evidence)
		return inventory, inventory.Validate()
	}

	inventory.RepositoryFacts.DefaultBranchProtected = Observed(branch.Protected, branchObs.Evidence)
	if branch.Protected {
		protection, protectionObs, err := reader.BranchProtection(ctx, fullName, branch.Name)
		if err != nil {
			return Inventory{}, err
		}
		inventory.Evidence = append(inventory.Evidence, protectionObs.Evidence)
		if protectionObs.Available {
			inventory.RepositoryFacts.RequiredStatusChecks = Observed(sortedUnique(protection.RequiredStatusChecks), protectionObs.Evidence)
			inventory.RepositoryFacts.RequiredReviews = Observed(protection.RequiredReviews, protectionObs.Evidence)
			if protection.AllowForcePushes == nil {
				inventory.RepositoryFacts.ForcePushProtected = Unknown[bool](protectionObs.Evidence)
			} else {
				inventory.RepositoryFacts.ForcePushProtected = Observed(!*protection.AllowForcePushes, protectionObs.Evidence)
			}
			if protection.AllowDeletions == nil {
				inventory.RepositoryFacts.DeletionProtected = Unknown[bool](protectionObs.Evidence)
			} else {
				inventory.RepositoryFacts.DeletionProtected = Observed(!*protection.AllowDeletions, protectionObs.Evidence)
			}
		} else {
			inventory.RepositoryFacts.RequiredStatusChecks = Unavailable[[]string](protectionObs.Evidence)
			inventory.RepositoryFacts.RequiredReviews = Unavailable[int](protectionObs.Evidence)
			inventory.RepositoryFacts.ForcePushProtected = Unavailable[bool](protectionObs.Evidence)
			inventory.RepositoryFacts.DeletionProtected = Unavailable[bool](protectionObs.Evidence)
		}
	} else {
		inventory.RepositoryFacts.RequiredStatusChecks = Observed([]string{}, branchObs.Evidence)
		inventory.RepositoryFacts.RequiredReviews = Observed(0, branchObs.Evidence)
		inventory.RepositoryFacts.ForcePushProtected = Observed(false, branchObs.Evidence)
		inventory.RepositoryFacts.DeletionProtected = Observed(false, branchObs.Evidence)
	}

	tree, treeObs, err := reader.Tree(ctx, fullName, branch.TreeSHA)
	if err != nil {
		return Inventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, treeObs.Evidence)
	if !treeObs.Available {
		setFileFactsUnavailable(&inventory, treeObs.Evidence)
		inventory.WorkflowsState = StateUnavailable
		return inventory, inventory.Validate()
	}
	populateTreeFacts(&inventory, tree, treeObs.Evidence)
	if err := populateWorkflows(ctx, reader, fullName, tree, &inventory); err != nil {
		return Inventory{}, err
	}
	return inventory, inventory.Validate()
}

func setRepositoryUnavailable(inventory *Inventory, evidence Evidence) {
	inventory.RepositoryFacts.DefaultBranch = Unavailable[string](evidence)
	setBranchUnavailable(inventory, evidence)
}

func setBranchUnavailable(inventory *Inventory, evidence Evidence) {
	inventory.RepositoryFacts.DefaultBranchProtected = Unavailable[bool](evidence)
	inventory.RepositoryFacts.RequiredStatusChecks = Unavailable[[]string](evidence)
	inventory.RepositoryFacts.RequiredReviews = Unavailable[int](evidence)
	inventory.RepositoryFacts.ForcePushProtected = Unavailable[bool](evidence)
	inventory.RepositoryFacts.DeletionProtected = Unavailable[bool](evidence)
	setFileFactsUnavailable(inventory, evidence)
	inventory.WorkflowsState = StateUnavailable
}

func setFileFactsUnavailable(inventory *Inventory, evidence Evidence) {
	inventory.RepositoryFacts.CODEOWNERSPresent = Unavailable[bool](evidence)
	inventory.RepositoryFacts.SecurityMDPresent = Unavailable[bool](evidence)
	inventory.RepositoryFacts.LicensePresent = Unavailable[bool](evidence)
	inventory.RepositoryFacts.IssueTemplatePresent = Unavailable[bool](evidence)
	inventory.RepositoryFacts.PullRequestTemplatePresent = Unavailable[bool](evidence)
	inventory.RepositoryFacts.DependencyAutomation = Unavailable[[]string](evidence)
	inventory.RepositoryFacts.WorkflowPaths = Unavailable[[]string](evidence)
}

func populateTreeFacts(inventory *Inventory, tree GitHubTree, evidence Evidence) {
	blobPaths := make(map[string]GitHubTreeEntry)
	for _, entry := range tree.Entries {
		if entry.Type == "blob" {
			blobPaths[strings.ToLower(entry.Path)] = entry
		}
	}
	complete := !tree.Truncated
	inventory.RepositoryFacts.CODEOWNERSPresent = treePresence(blobPaths, complete, evidence,
		"codeowners", ".github/codeowners", "docs/codeowners")
	inventory.RepositoryFacts.SecurityMDPresent = treePresence(blobPaths, complete, evidence,
		"security.md", ".github/security.md", "docs/security.md")
	inventory.RepositoryFacts.LicensePresent = licensePresence(blobPaths, complete, evidence)
	inventory.RepositoryFacts.IssueTemplatePresent = prefixPresence(blobPaths, complete, evidence, ".github/issue_template/")
	inventory.RepositoryFacts.PullRequestTemplatePresent = pullRequestTemplatePresence(blobPaths, complete, evidence)
	inventory.RepositoryFacts.DependencyAutomation = dependencyAutomation(blobPaths, complete, evidence)

	workflows := workflowEntries(tree.Entries)
	paths := make([]string, 0, len(workflows))
	for _, entry := range workflows {
		paths = append(paths, entry.Path)
	}
	if complete {
		inventory.RepositoryFacts.WorkflowPaths = Observed(paths, evidence)
		inventory.WorkflowsState = StateObserved
	} else {
		inventory.RepositoryFacts.WorkflowPaths = Unknown[[]string](evidence)
		inventory.WorkflowsState = StateUnknown
	}
}

func treePresence(entries map[string]GitHubTreeEntry, complete bool, evidence Evidence, candidates ...string) Fact[bool] {
	for _, candidate := range candidates {
		if entry, ok := entries[strings.ToLower(candidate)]; ok {
			return Observed(true, Evidence{Source: "github.git_tree", Reference: entry.Path})
		}
	}
	if complete {
		return Observed(false, evidence)
	}
	return Unknown[bool](evidence)
}

func prefixPresence(entries map[string]GitHubTreeEntry, complete bool, evidence Evidence, prefix string) Fact[bool] {
	prefix = strings.ToLower(prefix)
	for path, entry := range entries {
		if strings.HasPrefix(path, prefix) {
			return Observed(true, Evidence{Source: "github.git_tree", Reference: entry.Path})
		}
	}
	if complete {
		return Observed(false, evidence)
	}
	return Unknown[bool](evidence)
}

func licensePresence(entries map[string]GitHubTreeEntry, complete bool, evidence Evidence) Fact[bool] {
	for path, entry := range entries {
		if strings.Contains(path, "/") {
			continue
		}
		name := strings.ToLower(path)
		if name == "license" || name == "license.md" || name == "license.txt" || strings.HasPrefix(name, "license-") || name == "copying" || name == "copying.md" || name == "copying.txt" {
			return Observed(true, Evidence{Source: "github.git_tree", Reference: entry.Path})
		}
	}
	if complete {
		return Observed(false, evidence)
	}
	return Unknown[bool](evidence)
}

func pullRequestTemplatePresence(entries map[string]GitHubTreeEntry, complete bool, evidence Evidence) Fact[bool] {
	for path, entry := range entries {
		lower := strings.ToLower(path)
		if lower == "pull_request_template.md" || lower == ".github/pull_request_template.md" || lower == "docs/pull_request_template.md" || strings.HasPrefix(lower, ".github/pull_request_template/") {
			return Observed(true, Evidence{Source: "github.git_tree", Reference: entry.Path})
		}
	}
	if complete {
		return Observed(false, evidence)
	}
	return Unknown[bool](evidence)
}

func dependencyAutomation(entries map[string]GitHubTreeEntry, complete bool, evidence Evidence) Fact[[]string] {
	found := []string{}
	for path := range entries {
		switch strings.ToLower(path) {
		case ".github/dependabot.yml", ".github/dependabot.yaml":
			found = append(found, "dependabot")
		case "renovate.json", ".github/renovate.json", ".renovaterc", ".renovaterc.json":
			found = append(found, "renovate")
		}
	}
	found = sortedUnique(found)
	if len(found) > 0 || complete {
		return Observed(found, evidence)
	}
	return Unknown[[]string](evidence)
}

func workflowEntries(entries []GitHubTreeEntry) []GitHubTreeEntry {
	out := []GitHubTreeEntry{}
	for _, entry := range entries {
		lower := strings.ToLower(entry.Path)
		if entry.Type == "blob" && strings.HasPrefix(lower, ".github/workflows/") && (strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml")) {
			out = append(out, entry)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func populateWorkflows(ctx context.Context, reader GitHubReader, fullName string, tree GitHubTree, inventory *Inventory) error {
	for _, entry := range workflowEntries(tree.Entries) {
		data, obs, err := reader.Blob(ctx, fullName, entry.SHA)
		if err != nil {
			return err
		}
		inventory.Evidence = append(inventory.Evidence, obs.Evidence)
		workflowEvidence := Evidence{Source: "github.workflow", Reference: entry.Path}
		if !obs.Available {
			workflowEvidence.Detail = obs.Evidence.Detail
			inventory.Workflows = append(inventory.Workflows, Workflow{
				Path: entry.Path, State: StateUnavailable, Permissions: Permissions{Scopes: []PermissionScope{}}, Jobs: []WorkflowJob{}, Evidence: []Evidence{workflowEvidence},
			})
			continue
		}
		workflow, err := parseWorkflow(fullName, entry.Path, data, workflowEvidence)
		if err != nil {
			workflowEvidence.Detail = "workflow YAML could not be normalized"
			inventory.Workflows = append(inventory.Workflows, Workflow{
				Path: entry.Path, State: StateUnknown, Permissions: Permissions{Scopes: []PermissionScope{}}, Jobs: []WorkflowJob{}, Evidence: []Evidence{workflowEvidence},
			})
			continue
		}
		inventory.Workflows = append(inventory.Workflows, workflow)
	}
	return nil
}
