package posture

import (
	"context"
	"path"
)

type GitHubCommitReader interface {
	Commits(context.Context, string, string, int) ([]GitHubCommitSummary, bool, ReadObservation, error)
	Commit(context.Context, string, string) (GitHubCommitDetail, ReadObservation, error)
	CommitPullRequests(context.Context, string, string) (int, bool, ReadObservation, error)
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

	// Pin the history query to the exact commit observed above. Using the branch
	// name here would allow a concurrent branch advance to mix two snapshots.
	summaries, truncated, commitsObs, err := commitReader.Commits(ctx, fullName, branch.CommitSHA, profile.HistoryLimit)
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
		count, complete, prObs, err := reader.CommitPullRequests(ctx, fullName, summary.SHA)
		if err != nil {
			return CommitHistoryFact{}, err
		}
		if !prObs.Available {
			association = Unavailable[int](prObs.Evidence)
		} else if complete {
			association = Observed(count, prObs.Evidence)
		} else {
			association = Unknown[int](evidenceWithDetail(prObs.Evidence, "pull-request association list may be incomplete; exact count cannot be established"))
		}
	}
	inferenceEvidence := unsupportedReviewEvidence(summary.SHA)
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
	inferenceEvidence := unsupportedReviewEvidence(sha)
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
		DirectToDefaultBranch:         Unknown[bool](inferenceEvidence),
		UnreviewedChange:              Unknown[bool](inferenceEvidence),
	}
}

func unsupportedReviewEvidence(sha string) Evidence {
	return Evidence{Source: "repora.scope", Reference: sha, Detail: "commit/PR association does not prove direct-push or review status"}
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
