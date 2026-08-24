package posture

import (
	"context"
	"testing"
)

type fakeCommitReader struct {
	summaries        []GitHubCommitSummary
	truncated        bool
	commitsObs       ReadObservation
	details          map[string]GitHubCommitDetail
	detailObs        map[string]ReadObservation
	pullRequests     map[string]int
	pullRequestObs   map[string]ReadObservation
	commitCalls      int
	pullRequestCalls int
}

func (f *fakeCommitReader) Commits(context.Context, string, string, int) ([]GitHubCommitSummary, bool, ReadObservation, error) {
	return append([]GitHubCommitSummary(nil), f.summaries...), f.truncated, f.commitsObs, nil
}

func (f *fakeCommitReader) Commit(_ context.Context, _ string, sha string) (GitHubCommitDetail, ReadObservation, error) {
	f.commitCalls++
	return f.details[sha], f.detailObs[sha], nil
}

func (f *fakeCommitReader) CommitPullRequests(_ context.Context, _ string, sha string) (int, ReadObservation, error) {
	f.pullRequestCalls++
	return f.pullRequests[sha], f.pullRequestObs[sha], nil
}

func TestCollectGitHubCommitsNormalizesBoundedFacts(t *testing.T) {
	profile := []byte(`kind: repora.posture-commits-profile
version: 1
history_limit: 2
sensitive_paths:
  - .github/workflows/*
  - SECURITY.md
file_count_threshold: 2
changed_lines_threshold: 100
inspect_pull_requests: true
`)
	repoReader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repos/acme/project"),
		branch:        GitHubBranch{Name: "main", CommitSHA: "head", TreeSHA: "tree"},
		branchObs:     available("github.branch", "repos/acme/project/branches/main"),
		tree: GitHubTree{Entries: []GitHubTreeEntry{
			{Path: commitProfilePath, Type: "blob", SHA: "profile"},
		}},
		treeObs: available("github.git_tree", "acme/project:tree"),
		blobs: map[string][]byte{"profile": profile},
		blobObs: map[string]ReadObservation{"profile": available("github.blob", "profile")},
	}
	commitReader := &fakeCommitReader{
		summaries:  []GitHubCommitSummary{{SHA: "a"}, {SHA: "b"}},
		truncated:  true,
		commitsObs: available("github.commits", "acme/project:main"),
		details: map[string]GitHubCommitDetail{
			"a": {SHA: "a", ParentCount: 1, Verified: true, Additions: 80, Deletions: 30, Files: []string{"README.md", "SECURITY.md", ".github/workflows/ci.yml"}, FilesComplete: true},
			"b": {SHA: "b", ParentCount: 2, VerifyReason: "unsigned", Additions: 2, Deletions: 1, Files: []string{"docs/a.md"}, FilesComplete: true},
		},
		detailObs: map[string]ReadObservation{
			"a": available("github.commit", "a"),
			"b": available("github.commit", "b"),
		},
		pullRequests: map[string]int{"a": 1, "b": 0},
		pullRequestObs: map[string]ReadObservation{
			"a": available("github.commit_pulls", "a"),
			"b": available("github.commit_pulls", "b"),
		},
	}

	inventory, err := CollectGitHubCommits(context.Background(), repoReader, commitReader, "acme/project")
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if err := inventory.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if inventory.HistoryTruncated.Value == nil || !*inventory.HistoryTruncated.Value {
		t.Fatalf("history truncated = %#v", inventory.HistoryTruncated)
	}
	if len(inventory.Commits) != 2 {
		t.Fatalf("commits = %d", len(inventory.Commits))
	}
	first := inventory.Commits[0]
	if first.SignatureVerification.Value == nil || *first.SignatureVerification.Value != "verified" {
		t.Fatalf("signature = %#v", first.SignatureVerification)
	}
	if first.FileCountThresholdExceeded.Value == nil || !*first.FileCountThresholdExceeded.Value {
		t.Fatalf("file threshold = %#v", first.FileCountThresholdExceeded)
	}
	if first.ChangedLinesThresholdExceeded.Value == nil || !*first.ChangedLinesThresholdExceeded.Value {
		t.Fatalf("line threshold = %#v", first.ChangedLinesThresholdExceeded)
	}
	if first.SensitivePathsChanged.Value == nil || len(*first.SensitivePathsChanged.Value) != 2 {
		t.Fatalf("sensitive paths = %#v", first.SensitivePathsChanged)
	}
	if first.AssociatedPullRequests.Value == nil || *first.AssociatedPullRequests.Value != 1 {
		t.Fatalf("associated prs = %#v", first.AssociatedPullRequests)
	}
	if first.DirectToDefaultBranch.State != StateUnknown || first.UnreviewedChange.State != StateUnknown {
		t.Fatalf("unsupported review inferences = %#v %#v", first.DirectToDefaultBranch, first.UnreviewedChange)
	}
	second := inventory.Commits[1]
	if second.MergeCommit.Value == nil || !*second.MergeCommit.Value {
		t.Fatalf("merge fact = %#v", second.MergeCommit)
	}
	if second.SignatureVerification.Value == nil || *second.SignatureVerification.Value != "unsigned" {
		t.Fatalf("unsigned signature = %#v", second.SignatureVerification)
	}
}

func TestCommitFileIncompletenessPreservesUnknownNegativeFacts(t *testing.T) {
	profile := defaultCommitProfile()
	profile.SensitivePaths = []string{"SECURITY.md"}
	profile.FileCountThreshold = 5
	reader := &fakeCommitReader{
		details: map[string]GitHubCommitDetail{
			"a": {SHA: "a", ParentCount: 1, VerifyReason: "unsigned", Files: []string{"README.md"}, FilesComplete: false},
		},
		detailObs: map[string]ReadObservation{"a": available("github.commit", "a")},
		pullRequests: map[string]int{"a": 0},
		pullRequestObs: map[string]ReadObservation{"a": available("github.commit_pulls", "a")},
	}
	fact, err := collectCommitFact(context.Background(), reader, "acme/project", profile, GitHubCommitSummary{SHA: "a"})
	if err != nil {
		t.Fatalf("collect fact: %v", err)
	}
	if fact.FileCountThresholdExceeded.State != StateUnknown {
		t.Fatalf("file threshold = %#v", fact.FileCountThresholdExceeded)
	}
	if fact.SensitivePathsChanged.State != StateUnknown {
		t.Fatalf("sensitive paths = %#v", fact.SensitivePathsChanged)
	}
}

func TestCommitPositiveSensitiveMatchSurvivesIncompleteFileList(t *testing.T) {
	profile := defaultCommitProfile()
	profile.SensitivePaths = []string{"SECURITY.md"}
	reader := &fakeCommitReader{
		details: map[string]GitHubCommitDetail{
			"a": {SHA: "a", VerifyReason: "unsigned", Files: []string{"SECURITY.md"}, FilesComplete: false},
		},
		detailObs: map[string]ReadObservation{"a": available("github.commit", "a")},
		pullRequests: map[string]int{"a": 0},
		pullRequestObs: map[string]ReadObservation{"a": available("github.commit_pulls", "a")},
	}
	fact, err := collectCommitFact(context.Background(), reader, "acme/project", profile, GitHubCommitSummary{SHA: "a"})
	if err != nil {
		t.Fatalf("collect fact: %v", err)
	}
	if fact.SensitivePathsChanged.State != StateObserved || fact.SensitivePathsChanged.Value == nil || len(*fact.SensitivePathsChanged.Value) != 1 {
		t.Fatalf("sensitive paths = %#v", fact.SensitivePathsChanged)
	}
}
