package posture

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	CommitInventoryKind    = "repora.posture-commits"
	CommitInventoryVersion = 1
	CommitProfileKind      = "repora.posture-commits-profile"
	CommitProfileVersion   = 1
	commitProfilePath      = ".repora/posture-commits.yaml"
	defaultCommitLimit     = 20
	maxCommitLimit         = 50
	defaultFileThreshold   = 50
	defaultLineThreshold   = 1000
	maxCommitProfileBytes  = 1 << 20
	maxSensitivePatterns   = 128
	commitFilePageSize     = 100
)

type CommitProfile struct {
	Kind                  string   `json:"kind" yaml:"kind"`
	Version               int      `json:"version" yaml:"version"`
	HistoryLimit          int      `json:"history_limit" yaml:"history_limit"`
	SensitivePaths        []string `json:"sensitive_paths" yaml:"sensitive_paths"`
	FileCountThreshold    int      `json:"file_count_threshold" yaml:"file_count_threshold"`
	ChangedLinesThreshold int      `json:"changed_lines_threshold" yaml:"changed_lines_threshold"`
	InspectPullRequests   bool     `json:"inspect_pull_requests" yaml:"inspect_pull_requests"`
}

type GitHubCommitSummary struct {
	SHA string
}

type GitHubCommitDetail struct {
	SHA          string
	ParentCount  int
	Verified     bool
	VerifyReason string
	Additions    int
	Deletions    int
	Files        []string
	FilesComplete bool
}

type GitHubCommitReader interface {
	Commits(context.Context, string, string, int) ([]GitHubCommitSummary, bool, ReadObservation, error)
	Commit(context.Context, string, string) (GitHubCommitDetail, ReadObservation, error)
	CommitPullRequests(context.Context, string, string) (int, ReadObservation, error)
}

type CommitHistoryFact struct {
	SHA                           string         `json:"sha"`
	MergeCommit                   Fact[bool]     `json:"merge_commit"`
	SignatureVerification         Fact[string]   `json:"signature_verification"`
	ChangedLines                  Fact[int]      `json:"changed_lines"`
	ObservedFileCount             Fact[int]      `json:"observed_file_count"`
	FilesComplete                 Fact[bool]     `json:"files_complete"`
	FileCountThresholdExceeded    Fact[bool]     `json:"file_count_threshold_exceeded"`
	ChangedLinesThresholdExceeded Fact[bool]     `json:"changed_lines_threshold_exceeded"`
	SensitivePathsChanged         Fact[[]string] `json:"sensitive_paths_changed"`
	AssociatedPullRequests        Fact[int]      `json:"associated_pull_requests"`
	DirectToDefaultBranch         Fact[bool]     `json:"direct_to_default_branch"`
	UnreviewedChange              Fact[bool]     `json:"unreviewed_change"`
}

type CommitInventory struct {
	Kind                       string               `json:"kind"`
	Version                    int                  `json:"version"`
	Repository                 RepositoryIdentity   `json:"repository"`
	DefaultBranch              Fact[string]         `json:"default_branch"`
	DefaultCommit              Fact[string]         `json:"default_commit"`
	ProfileDeclared            Fact[bool]           `json:"profile_declared"`
	HistoryLimit               Fact[int]            `json:"history_limit"`
	HistoryTruncated           Fact[bool]           `json:"history_truncated"`
	FileCountThreshold         Fact[int]            `json:"file_count_threshold"`
	ChangedLinesThreshold      Fact[int]            `json:"changed_lines_threshold"`
	SensitivePathPatterns      Fact[[]string]       `json:"sensitive_path_patterns"`
	Commits                    []CommitHistoryFact  `json:"commits"`
	SignedTagCount             Fact[int]            `json:"signed_tag_count"`
	UnsignedTagCount           Fact[int]            `json:"unsigned_tag_count"`
	ReleaseBoundaryChangeCount Fact[int]            `json:"release_boundary_change_count"`
	Evidence                   []Evidence           `json:"evidence"`
}

func defaultCommitProfile() CommitProfile {
	return CommitProfile{
		Kind:                  CommitProfileKind,
		Version:               CommitProfileVersion,
		HistoryLimit:          defaultCommitLimit,
		SensitivePaths:        []string{},
		FileCountThreshold:    defaultFileThreshold,
		ChangedLinesThreshold: defaultLineThreshold,
		InspectPullRequests:   true,
	}
}

func ParseCommitProfile(data []byte) (CommitProfile, error) {
	if len(data) > maxCommitProfileBytes {
		return CommitProfile{}, fmt.Errorf("commit posture profile exceeds %d bytes", maxCommitProfileBytes)
	}
	profile := defaultCommitProfile()
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&profile); err != nil {
		return CommitProfile{}, fmt.Errorf("parse commit posture profile: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return CommitProfile{}, fmt.Errorf("commit posture profile must contain exactly one YAML document")
		}
		return CommitProfile{}, fmt.Errorf("parse commit posture profile trailing content: %w", err)
	}
	if err := profile.Validate(); err != nil {
		return CommitProfile{}, err
	}
	profile.SensitivePaths = sortedUnique(profile.SensitivePaths)
	return profile, nil
}

func (p CommitProfile) Validate() error {
	if p.Kind != CommitProfileKind || p.Version != CommitProfileVersion {
		return fmt.Errorf("unsupported commit posture profile contract: kind=%q version=%d", p.Kind, p.Version)
	}
	if p.HistoryLimit < 1 || p.HistoryLimit > maxCommitLimit {
		return fmt.Errorf("history_limit must be between 1 and %d", maxCommitLimit)
	}
	if p.FileCountThreshold < 1 || p.ChangedLinesThreshold < 1 {
		return fmt.Errorf("commit thresholds must be positive")
	}
	if len(p.SensitivePaths) > maxSensitivePatterns {
		return fmt.Errorf("sensitive_paths exceeds %d patterns", maxSensitivePatterns)
	}
	for _, pattern := range p.SensitivePaths {
		if err := validateSensitivePattern(pattern); err != nil {
			return err
		}
	}
	return nil
}

func validateSensitivePattern(pattern string) error {
	if strings.TrimSpace(pattern) == "" || strings.HasPrefix(pattern, "/") || strings.Contains(pattern, "\\") {
		return fmt.Errorf("sensitive path pattern must be a non-empty repository-relative slash path: %q", pattern)
	}
	for _, part := range strings.Split(pattern, "/") {
		if part == ".." {
			return fmt.Errorf("sensitive path pattern must not traverse parents: %q", pattern)
		}
	}
	if _, err := path.Match(pattern, "probe"); err != nil {
		return fmt.Errorf("invalid sensitive path pattern %q: %w", pattern, err)
	}
	return nil
}

func newCommitInventory(fullName string) CommitInventory {
	scopeEvidence := Evidence{Source: "repora.scope", Reference: "posture-commits-v1", Detail: "tag signatures and release boundaries are not inspected by commit posture v1"}
	return CommitInventory{
		Kind:                       CommitInventoryKind,
		Version:                    CommitInventoryVersion,
		Repository:                 RepositoryIdentity{Provider: "github", FullName: fullName},
		Commits:                    []CommitHistoryFact{},
		SignedTagCount:             Unknown[int](scopeEvidence),
		UnsignedTagCount:           Unknown[int](scopeEvidence),
		ReleaseBoundaryChangeCount: Unknown[int](scopeEvidence),
		Evidence:                   []Evidence{scopeEvidence},
	}
}

func (i CommitInventory) Marshal() ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode commit posture inventory: %w", err)
	}
	return append(data, '\n'), nil
}

func (i CommitInventory) Validate() error {
	if i.Kind != CommitInventoryKind || i.Version != CommitInventoryVersion {
		return fmt.Errorf("unsupported commit posture inventory contract: kind=%q version=%d", i.Kind, i.Version)
	}
	if i.Repository.Provider != "github" {
		return fmt.Errorf("commit posture inventory provider must be github")
	}
	if _, _, err := splitGitHubFullName(i.Repository.FullName); err != nil {
		return err
	}
	checks := []error{
		validateFact("default_branch", i.DefaultBranch),
		validateFact("default_commit", i.DefaultCommit),
		validateFact("profile_declared", i.ProfileDeclared),
		validateFact("history_limit", i.HistoryLimit),
		validateFact("history_truncated", i.HistoryTruncated),
		validateFact("file_count_threshold", i.FileCountThreshold),
		validateFact("changed_lines_threshold", i.ChangedLinesThreshold),
		validateFact("sensitive_path_patterns", i.SensitivePathPatterns),
		validateFact("signed_tag_count", i.SignedTagCount),
		validateFact("unsigned_tag_count", i.UnsignedTagCount),
		validateFact("release_boundary_change_count", i.ReleaseBoundaryChangeCount),
	}
	for _, err := range checks {
		if err != nil {
			return err
		}
	}
	if i.Commits == nil || i.Evidence == nil {
		return fmt.Errorf("commit posture arrays are required")
	}
	for idx, commit := range i.Commits {
		if strings.TrimSpace(commit.SHA) == "" {
			return fmt.Errorf("commits[%d] sha is required", idx)
		}
		for name, err := range map[string]error{
			"merge_commit": validateFact("merge_commit", commit.MergeCommit),
			"signature_verification": validateFact("signature_verification", commit.SignatureVerification),
			"changed_lines": validateFact("changed_lines", commit.ChangedLines),
			"observed_file_count": validateFact("observed_file_count", commit.ObservedFileCount),
			"files_complete": validateFact("files_complete", commit.FilesComplete),
			"file_count_threshold_exceeded": validateFact("file_count_threshold_exceeded", commit.FileCountThresholdExceeded),
			"changed_lines_threshold_exceeded": validateFact("changed_lines_threshold_exceeded", commit.ChangedLinesThresholdExceeded),
			"sensitive_paths_changed": validateFact("sensitive_paths_changed", commit.SensitivePathsChanged),
			"associated_pull_requests": validateFact("associated_pull_requests", commit.AssociatedPullRequests),
			"direct_to_default_branch": validateFact("direct_to_default_branch", commit.DirectToDefaultBranch),
			"unreviewed_change": validateFact("unreviewed_change", commit.UnreviewedChange),
		} {
			if err != nil {
				return fmt.Errorf("commits[%d] %s: %w", idx, name, err)
			}
		}
	}
	return nil
}

func CollectGitHubCommits(ctx context.Context, repoReader GitHubReader, commitReader GitHubCommitReader, fullName string) (CommitInventory, error) {
	if _, _, err := splitGitHubFullName(fullName); err != nil {
		return CommitInventory{}, err
	}
	inventory := newCommitInventory(fullName)
	repo, repoObs, err := repoReader.Repository(ctx, fullName)
	if err != nil {
		return CommitInventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, repoObs.Evidence)
	if !repoObs.Available {
		setCommitInventoryUnavailable(&inventory, repoObs.Evidence)
		return inventory, inventory.Validate()
	}
	inventory.DefaultBranch = Observed(repo.DefaultBranch, repoObs.Evidence)
	branch, branchObs, err := repoReader.Branch(ctx, fullName, repo.DefaultBranch)
	if err != nil {
		return CommitInventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, branchObs.Evidence)
	if !branchObs.Available {
		setCommitInventoryAfterBranchUnavailable(&inventory, branchObs.Evidence)
		return inventory, inventory.Validate()
	}
	inventory.DefaultCommit = Observed(branch.CommitSHA, branchObs.Evidence)
	tree, treeObs, err := repoReader.Tree(ctx, fullName, branch.TreeSHA)
	if err != nil {
		return CommitInventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, treeObs.Evidence)
	if !treeObs.Available {
		setCommitInventoryAfterTreeUnavailable(&inventory, treeObs.Evidence)
		return inventory, inventory.Validate()
	}
	entries := make(map[string]GitHubTreeEntry, len(tree.Entries))
	for _, entry := range tree.Entries {
		entries[entry.Path] = entry
	}
	profile, profileState, profileEvidence, err := loadCommitProfile(ctx, repoReader, fullName, tree, entries, treeObs.Evidence)
	if err != nil {
		return CommitInventory{}, err
	}
	inventory.ProfileDeclared = presenceFact(entries, tree, commitProfilePath, treeObs.Evidence)
	if profileState != StateObserved {
		inventory.HistoryLimit = factForState[int](profileState, profileEvidence)
		inventory.HistoryTruncated = factForState[bool](profileState, profileEvidence)
		inventory.FileCountThreshold = factForState[int](profileState, profileEvidence)
		inventory.ChangedLinesThreshold = factForState[int](profileState, profileEvidence)
		inventory.SensitivePathPatterns = factForState[[]string](profileState, profileEvidence)
		return inventory, inventory.Validate()
	}
	inventory.HistoryLimit = Observed(profile.HistoryLimit, profileEvidence)
	inventory.FileCountThreshold = Observed(profile.FileCountThreshold, profileEvidence)
	inventory.ChangedLinesThreshold = Observed(profile.ChangedLinesThreshold, profileEvidence)
	inventory.SensitivePathPatterns = Observed(profile.SensitivePaths, profileEvidence)

	summaries, truncated, commitsObs, err := commitReader.Commits(ctx, fullName, repo.DefaultBranch, profile.HistoryLimit)
	if err != nil {
		return CommitInventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, commitsObs.Evidence)
	if !commitsObs.Available {
		inventory.HistoryTruncated = Unavailable[bool](commitsObs.Evidence)
		return inventory, inventory.Validate()
	}
	inventory.HistoryTruncated = Observed(truncated, commitsObs.Evidence)
	for _, summary := range summaries {
		fact, err := collectCommitFact(ctx, commitReader, fullName, profile, summary)
		if err != nil {
			return CommitInventory{}, err
		}
		inventory.Commits = append(inventory.Commits, fact)
	}
	return inventory, inventory.Validate()
}

func collectCommitFact(ctx context.Context, reader GitHubCommitReader, fullName string, profile CommitProfile, summary GitHubCommitSummary) (CommitHistoryFact, error) {
	detail, obs, err := reader.Commit(ctx, fullName, summary.SHA)
	if err != nil {
		return CommitHistoryFact{}, err
	}
	if !obs.Available {
		return unavailableCommitFact(summary.SHA, obs.Evidence), nil
	}
	files := sortedUnique(detail.Files)
	changedLines := detail.Additions + detail.Deletions
	fileThreshold := thresholdFact(len(files), profile.FileCountThreshold, detail.FilesComplete, obs.Evidence)
	sensitive := sensitivePathMatches(files, profile.SensitivePaths)
	sensitiveFact := Observed(sensitive, obs.Evidence)
	if !detail.FilesComplete && len(sensitive) == 0 && len(profile.SensitivePaths) > 0 {
		sensitiveFact = Unknown[[]string](evidenceWithDetail(obs.Evidence, "commit file list may be incomplete; sensitive-path absence cannot be established"))
	}
	association := Unknown[int](Evidence{Source: "repora.scope", Reference: summary.SHA, Detail: "pull-request association observation disabled by profile"})
	if profile.InspectPullRequests {
		count, prObs, err := reader.CommitPullRequests(ctx, fullName, summary.SHA)
		if err != nil {
			return CommitHistoryFact{}, err
		}
		if prObs.Available {
			association = Observed(count, prObs.Evidence)
		} else {
			association = Unavailable[int](prObs.Evidence)
		}
	}
	inferenceEvidence := Evidence{Source: "repora.scope", Reference: summary.SHA, Detail: "commit/PR association does not prove direct-push or review status"}
	return CommitHistoryFact{
		SHA:                           detail.SHA,
		MergeCommit:                   Observed(detail.ParentCount > 1, obs.Evidence),
		SignatureVerification:         signatureVerificationFact(detail, obs.Evidence),
		ChangedLines:                  Observed(changedLines, obs.Evidence),
		ObservedFileCount:             Observed(len(files), obs.Evidence),
		FilesComplete:                 Observed(detail.FilesComplete, obs.Evidence),
		FileCountThresholdExceeded:    fileThreshold,
		ChangedLinesThresholdExceeded: Observed(changedLines > profile.ChangedLinesThreshold, obs.Evidence),
		SensitivePathsChanged:         sensitiveFact,
		AssociatedPullRequests:        association,
		DirectToDefaultBranch:         Unknown[bool](inferenceEvidence),
		UnreviewedChange:              Unknown[bool](inferenceEvidence),
	}, nil
}

func unavailableCommitFact(sha string, evidence Evidence) CommitHistoryFact {
	return CommitHistoryFact{
		SHA:                           sha,
		MergeCommit:                   Unavailable[bool](evidence),
		SignatureVerification:         Unavailable[string](evidence),
		ChangedLines:                  Unavailable[int](evidence),
		ObservedFileCount:             Unavailable[int](evidence),
		FilesComplete:                 Unavailable[bool](evidence),
		FileCountThresholdExceeded:    Unavailable[bool](evidence),
		ChangedLinesThresholdExceeded: Unavailable[bool](evidence),
		SensitivePathsChanged:         Unavailable[[]string](evidence),
		AssociatedPullRequests:        Unavailable[int](evidence),
		DirectToDefaultBranch:         Unavailable[bool](evidence),
		UnreviewedChange:              Unavailable[bool](evidence),
	}
}

func thresholdFact(observed, threshold int, complete bool, evidence Evidence) Fact[bool] {
	if observed > threshold {
		return Observed(true, evidence)
	}
	if complete {
		return Observed(false, evidence)
	}
	return Unknown[bool](evidenceWithDetail(evidence, "commit file list may be incomplete; threshold absence cannot be established"))
}

func signatureVerificationFact(detail GitHubCommitDetail, evidence Evidence) Fact[string] {
	if detail.Verified {
		return Observed("verified", evidence)
	}
	switch detail.VerifyReason {
	case "unsigned":
		return Observed("unsigned", evidence)
	case "":
		return Unknown[string](evidenceWithDetail(evidence, "GitHub did not expose commit signature verification reason"))
	default:
		return Observed("unverified", evidenceWithDetail(evidence, "GitHub verification reason: "+detail.VerifyReason))
	}
}

func sensitivePathMatches(files, patterns []string) []string {
	matches := []string{}
	for _, file := range files {
		for _, pattern := range patterns {
			matched, _ := path.Match(pattern, file)
			if matched {
				matches = append(matches, file)
				break
			}
		}
	}
	return sortedUnique(matches)
}

func loadCommitProfile(ctx context.Context, reader GitHubReader, fullName string, tree GitHubTree, entries map[string]GitHubTreeEntry, treeEvidence Evidence) (CommitProfile, FactState, Evidence, error) {
	entry, ok := entries[commitProfilePath]
	if !ok {
		if tree.Truncated {
			evidence := evidenceWithDetail(treeEvidence, "Git tree is truncated; commit posture profile presence is unknown")
			return CommitProfile{}, StateUnknown, evidence, nil
		}
		return defaultCommitProfile(), StateObserved, Evidence{Source: "repora.builtin", Reference: "commit-profile:baseline"}, nil
	}
	if entry.Type != "blob" {
		return CommitProfile{}, StateUnknown, evidenceWithDetail(treeEvidence, "commit posture profile exists but is not a blob"), nil
	}
	data, obs, err := reader.Blob(ctx, fullName, entry.SHA)
	if err != nil {
		return CommitProfile{}, "", Evidence{}, err
	}
	if !obs.Available {
		return CommitProfile{}, StateUnavailable, obs.Evidence, nil
	}
	profile, err := ParseCommitProfile(data)
	if err != nil {
		return CommitProfile{}, StateUnknown, evidenceWithDetail(obs.Evidence, "declared commit posture profile is malformed or unsupported"), nil
	}
	return profile, StateObserved, obs.Evidence, nil
}

func setCommitInventoryUnavailable(inventory *CommitInventory, evidence Evidence) {
	inventory.DefaultBranch = Unavailable[string](evidence)
	inventory.DefaultCommit = Unavailable[string](evidence)
	inventory.ProfileDeclared = Unavailable[bool](evidence)
	inventory.HistoryLimit = Unavailable[int](evidence)
	inventory.HistoryTruncated = Unavailable[bool](evidence)
	inventory.FileCountThreshold = Unavailable[int](evidence)
	inventory.ChangedLinesThreshold = Unavailable[int](evidence)
	inventory.SensitivePathPatterns = Unavailable[[]string](evidence)
}

func setCommitInventoryAfterBranchUnavailable(inventory *CommitInventory, evidence Evidence) {
	inventory.DefaultCommit = Unavailable[string](evidence)
	inventory.ProfileDeclared = Unavailable[bool](evidence)
	inventory.HistoryLimit = Unavailable[int](evidence)
	inventory.HistoryTruncated = Unavailable[bool](evidence)
	inventory.FileCountThreshold = Unavailable[int](evidence)
	inventory.ChangedLinesThreshold = Unavailable[int](evidence)
	inventory.SensitivePathPatterns = Unavailable[[]string](evidence)
}

func setCommitInventoryAfterTreeUnavailable(inventory *CommitInventory, evidence Evidence) {
	inventory.ProfileDeclared = Unavailable[bool](evidence)
	inventory.HistoryLimit = Unavailable[int](evidence)
	inventory.HistoryTruncated = Unavailable[bool](evidence)
	inventory.FileCountThreshold = Unavailable[int](evidence)
	inventory.ChangedLinesThreshold = Unavailable[int](evidence)
	inventory.SensitivePathPatterns = Unavailable[[]string](evidence)
}

func (r *HTTPGitHubReader) Commits(ctx context.Context, fullName, branch string, limit int) ([]GitHubCommitSummary, bool, ReadObservation, error) {
	owner, repo, err := splitGitHubFullName(fullName)
	if err != nil {
		return nil, false, ReadObservation{}, err
	}
	if limit < 1 || limit > maxCommitLimit {
		return nil, false, ReadObservation{}, fmt.Errorf("commit history limit must be between 1 and %d", maxCommitLimit)
	}
	requestLimit := limit + 1
	apiPath := fmt.Sprintf("/repos/%s/%s/commits?sha=%s&per_page=%d", url.PathEscape(owner), url.PathEscape(repo), url.QueryEscape(branch), requestLimit)
	var response []struct {
		SHA string `json:"sha"`
	}
	obs, err := r.getJSON(ctx, apiPath, "github.commits", fmt.Sprintf("%s:%s", fullName, branch), &response)
	if err != nil || !obs.Available {
		return nil, false, obs, err
	}
	truncated := len(response) > limit
	if truncated {
		response = response[:limit]
	}
	out := make([]GitHubCommitSummary, 0, len(response))
	for _, item := range response {
		if item.SHA == "" {
			return nil, false, obs, fmt.Errorf("GitHub commit history returned an empty sha")
		}
		out = append(out, GitHubCommitSummary{SHA: item.SHA})
	}
	return out, truncated, obs, nil
}

func (r *HTTPGitHubReader) Commit(ctx context.Context, fullName, sha string) (GitHubCommitDetail, ReadObservation, error) {
	owner, repo, err := splitGitHubFullName(fullName)
	if err != nil {
		return GitHubCommitDetail{}, ReadObservation{}, err
	}
	apiPath := fmt.Sprintf("/repos/%s/%s/commits/%s?per_page=%d", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(sha), commitFilePageSize)
	var response struct {
		SHA     string `json:"sha"`
		Parents []struct {
			SHA string `json:"sha"`
		} `json:"parents"`
		Commit struct {
			Verification struct {
				Verified bool   `json:"verified"`
				Reason   string `json:"reason"`
			} `json:"verification"`
		} `json:"commit"`
		Stats struct {
			Additions int `json:"additions"`
			Deletions int `json:"deletions"`
		} `json:"stats"`
		Files []struct {
			Filename string `json:"filename"`
		} `json:"files"`
	}
	obs, err := r.getJSON(ctx, apiPath, "github.commit", fmt.Sprintf("%s:%s", fullName, sha), &response)
	if err != nil || !obs.Available {
		return GitHubCommitDetail{}, obs, err
	}
	if response.SHA == "" {
		return GitHubCommitDetail{}, obs, fmt.Errorf("GitHub commit %s returned an empty sha", sha)
	}
	files := make([]string, 0, len(response.Files))
	for _, file := range response.Files {
		if file.Filename != "" {
			files = append(files, file.Filename)
		}
	}
	return GitHubCommitDetail{
		SHA:           response.SHA,
		ParentCount:   len(response.Parents),
		Verified:      response.Commit.Verification.Verified,
		VerifyReason:  response.Commit.Verification.Reason,
		Additions:     response.Stats.Additions,
		Deletions:     response.Stats.Deletions,
		Files:         sortedUnique(files),
		FilesComplete: len(response.Files) < commitFilePageSize,
	}, obs, nil
}

func (r *HTTPGitHubReader) CommitPullRequests(ctx context.Context, fullName, sha string) (int, ReadObservation, error) {
	owner, repo, err := splitGitHubFullName(fullName)
	if err != nil {
		return 0, ReadObservation{}, err
	}
	apiPath := fmt.Sprintf("/repos/%s/%s/commits/%s/pulls?per_page=100", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(sha))
	var response []struct {
		Number int `json:"number"`
	}
	obs, err := r.getJSON(ctx, apiPath, "github.commit_pulls", fmt.Sprintf("%s:%s", fullName, sha), &response)
	if err != nil || !obs.Available {
		return 0, obs, err
	}
	return len(response), obs, nil
}

func sortedCommitFacts(facts []CommitHistoryFact) []CommitHistoryFact {
	out := append([]CommitHistoryFact(nil), facts...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].SHA < out[j].SHA })
	return out
}
