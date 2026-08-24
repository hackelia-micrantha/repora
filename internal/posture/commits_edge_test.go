package posture

import (
	"context"
	"strings"
	"testing"
)

func TestUnavailableCommitDetailDoesNotInventFacts(t *testing.T) {
	reader := &fakeCommitReader{
		details:        map[string]GitHubCommitDetail{},
		detailObs:      map[string]ReadObservation{"a": unavailable("github.commit", "a")},
		pullRequests:   map[string]int{},
		pullRequestObs: map[string]ReadObservation{},
	}
	fact, err := collectCommitFact(context.Background(), reader, "acme/project", defaultCommitProfile(), GitHubCommitSummary{SHA: "a"})
	if err != nil {
		t.Fatalf("collect fact: %v", err)
	}
	if fact.SignatureVerification.State != StateUnavailable || fact.ChangedLines.State != StateUnavailable || fact.SensitivePathsChanged.State != StateUnavailable {
		t.Fatalf("unavailable fact = %#v", fact)
	}
	if reader.pullRequestCalls != 0 {
		t.Fatalf("pull request calls = %d, want 0 after unavailable commit detail", reader.pullRequestCalls)
	}
}

func TestUnavailablePullRequestEvidenceDoesNotImplyUnreviewed(t *testing.T) {
	reader := &fakeCommitReader{
		details: map[string]GitHubCommitDetail{
			"a": {SHA: "a", VerifyReason: "unsigned", FilesComplete: true},
		},
		detailObs:      map[string]ReadObservation{"a": available("github.commit", "a")},
		pullRequests:   map[string]int{"a": 0},
		pullRequestObs: map[string]ReadObservation{"a": unavailable("github.commit_pulls", "a")},
	}
	fact, err := collectCommitFact(context.Background(), reader, "acme/project", defaultCommitProfile(), GitHubCommitSummary{SHA: "a"})
	if err != nil {
		t.Fatalf("collect fact: %v", err)
	}
	if fact.AssociatedPullRequests.State != StateUnavailable {
		t.Fatalf("associated prs = %#v", fact.AssociatedPullRequests)
	}
	if fact.DirectToDefaultBranch.State != StateUnknown || fact.UnreviewedChange.State != StateUnknown {
		t.Fatalf("unsupported review inference = %#v %#v", fact.DirectToDefaultBranch, fact.UnreviewedChange)
	}
}

func TestMalformedCommitProfilePreservesUnknownDependentFacts(t *testing.T) {
	repoReader := fakeGitHubReader{
		repository:    GitHubRepository{DefaultBranch: "main"},
		repositoryObs: available("github.repository", "repo"),
		branch:        GitHubBranch{Name: "main", CommitSHA: "head", TreeSHA: "tree"},
		branchObs:     available("github.branch", "branch"),
		tree:          GitHubTree{Entries: []GitHubTreeEntry{{Path: commitProfilePath, Type: "blob", SHA: "profile"}}},
		treeObs:       available("github.git_tree", "tree"),
		blobs:         map[string][]byte{"profile": []byte("kind: wrong\nversion: 1\n")},
		blobObs:       map[string]ReadObservation{"profile": available("github.blob", "profile")},
	}
	commitReader := &fakeCommitReader{}
	inventory, err := CollectGitHubCommits(context.Background(), repoReader, commitReader, "acme/project")
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if inventory.ProfileDeclared.State != StateObserved || inventory.ProfileDeclared.Value == nil || !*inventory.ProfileDeclared.Value {
		t.Fatalf("profile declared = %#v", inventory.ProfileDeclared)
	}
	if inventory.HistoryLimit.State != StateUnknown || inventory.FileCountThreshold.State != StateUnknown {
		t.Fatalf("dependent facts = %#v %#v", inventory.HistoryLimit, inventory.FileCountThreshold)
	}
	if commitReader.commitCalls != 0 {
		t.Fatalf("commit detail calls = %d, want 0", commitReader.commitCalls)
	}
}

func TestCommitProfileRejectsUnsafeOrUnboundedConfiguration(t *testing.T) {
	for _, data := range []string{
		"kind: repora.posture-commits-profile\nversion: 1\nhistory_limit: 51\nfile_count_threshold: 1\nchanged_lines_threshold: 1\n",
		"kind: repora.posture-commits-profile\nversion: 1\nhistory_limit: 1\nsensitive_paths: [../secret]\nfile_count_threshold: 1\nchanged_lines_threshold: 1\n",
	} {
		if _, err := ParseCommitProfile([]byte(data)); err == nil {
			t.Fatalf("ParseCommitProfile(%q) succeeded", data)
		}
	}
}

func TestCommitInventoryContainsNoIdentityOrProductivityFields(t *testing.T) {
	inventory := newCommitInventory("acme/project")
	inventory.DefaultBranch = Observed("main")
	inventory.DefaultCommit = Observed("abc")
	inventory.ProfileDeclared = Observed(false)
	inventory.HistoryLimit = Observed(20)
	inventory.HistoryTruncated = Observed(false)
	inventory.FileCountThreshold = Observed(50)
	inventory.ChangedLinesThreshold = Observed(1000)
	inventory.SensitivePathPatterns = Observed([]string{})
	data, err := inventory.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	text := string(data)
	for _, forbidden := range []string{"author", "committer", "productivity", "performance_score", "intent"} {
		if containsJSONKey(text, forbidden) {
			t.Fatalf("serialized inventory contains forbidden field %q: %s", forbidden, text)
		}
	}
}

func containsJSONKey(text, key string) bool {
	return strings.Contains(text, `"`+key+`"`)
}
