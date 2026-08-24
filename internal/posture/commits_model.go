package posture

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
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
	SHA           string
	ParentCount   int
	Verified      bool
	VerifyReason  string
	Additions     int
	Deletions     int
	Files         []string
	FilesComplete bool
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
	Kind                       string              `json:"kind"`
	Version                    int                 `json:"version"`
	Repository                 RepositoryIdentity  `json:"repository"`
	DefaultBranch              Fact[string]        `json:"default_branch"`
	DefaultCommit              Fact[string]        `json:"default_commit"`
	ProfileDeclared            Fact[bool]          `json:"profile_declared"`
	HistoryLimit               Fact[int]           `json:"history_limit"`
	HistoryTruncated           Fact[bool]          `json:"history_truncated"`
	FileCountThreshold         Fact[int]           `json:"file_count_threshold"`
	ChangedLinesThreshold      Fact[int]           `json:"changed_lines_threshold"`
	SensitivePathPatterns      Fact[[]string]      `json:"sensitive_path_patterns"`
	Commits                    []CommitHistoryFact `json:"commits"`
	SignedTagCount             Fact[int]           `json:"signed_tag_count"`
	UnsignedTagCount           Fact[int]           `json:"unsigned_tag_count"`
	ReleaseBoundaryChangeCount Fact[int]           `json:"release_boundary_change_count"`
	Evidence                   []Evidence          `json:"evidence"`
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
		checks := []error{
			validateFact("merge_commit", commit.MergeCommit),
			validateFact("signature_verification", commit.SignatureVerification),
			validateFact("changed_lines", commit.ChangedLines),
			validateFact("observed_file_count", commit.ObservedFileCount),
			validateFact("files_complete", commit.FilesComplete),
			validateFact("file_count_threshold_exceeded", commit.FileCountThresholdExceeded),
			validateFact("changed_lines_threshold_exceeded", commit.ChangedLinesThresholdExceeded),
			validateFact("sensitive_paths_changed", commit.SensitivePathsChanged),
			validateFact("associated_pull_requests", commit.AssociatedPullRequests),
			validateFact("direct_to_default_branch", commit.DirectToDefaultBranch),
			validateFact("unreviewed_change", commit.UnreviewedChange),
		}
		for _, err := range checks {
			if err != nil {
				return fmt.Errorf("commits[%d]: %w", idx, err)
			}
		}
	}
	return nil
}
