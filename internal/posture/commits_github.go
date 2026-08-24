package posture

import (
	"context"
	"fmt"
	"net/url"
)

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
