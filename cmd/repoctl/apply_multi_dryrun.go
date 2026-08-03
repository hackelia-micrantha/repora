package main

import (
	"fmt"
	"os"
	"sync"

	"repoctl/internal/apply"
	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

type multiDryRunTaskResult struct {
	index  int
	result apply.Result
	err    error
}

var repositoryPreflightExecute = func(repo config.Repo, observed status.RepositoryResult, artifact planartifact.Artifact, audit apply.Audit) (apply.Result, error) {
	return apply.PreflightRepositoryArtifactAudited(repo, observed, artifact, gitwrap.Client{}, audit)
}

func runMultiMirrorDryRun(spec config.Spec, parallel int, jsonFlag bool, artifact *planartifact.Artifact, configPath string, debug bool) int {
	if jsonFlag {
		fmt.Fprintln(os.Stderr, "repoctl: multi-mirror apply --dry-run --json is not supported until the per-target apply contract is versioned")
		return 1
	}
	if artifact != nil && len(artifact.Repositories) != len(spec.Repos) {
		fmt.Fprintf(os.Stderr, "repoctl: plan artifact contains %d repositories for %d selected configuration repositories\n", len(artifact.Repositories), len(spec.Repos))
		return 1
	}
	audit, err := newAudit(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: initialize execution journal: %v\n", err)
		return 1
	}

	sem := make(chan struct{}, parallel)
	resultsCh := make(chan multiDryRunTaskResult, len(spec.Repos))
	var wg sync.WaitGroup
	wg.Add(len(spec.Repos))
	for i, repo := range spec.Repos {
		i := i
		repo := repo
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if debug {
				debugf("repoctl: debug preflighting repo=%s mirrors=%d\n", repo.ID, len(repo.Mirrors))
			}

			if len(repo.Mirrors) == 1 {
				observed, err := statusCheck(repo)
				if err != nil {
					resultsCh <- multiDryRunTaskResult{index: i, err: err}
					return
				}
				var single planartifact.Artifact
				if artifact != nil {
					single = oneRepositoryArtifact(*artifact, i)
				} else {
					single, err = planBuild(repo, observed, true)
					if err != nil {
						resultsCh <- multiDryRunTaskResult{index: i, err: err}
						return
					}
				}
				result, err := auditedArtifactApplyExecute(repo, observed, single, false, true, *audit)
				resultsCh <- multiDryRunTaskResult{index: i, result: result, err: err}
				return
			}

			observed, err := statusCheckAll(repo)
			if err != nil {
				resultsCh <- multiDryRunTaskResult{index: i, err: err}
				return
			}
			var single planartifact.Artifact
			if artifact != nil {
				single = oneRepositoryArtifact(*artifact, i)
			} else {
				single, err = repositoryPlanBuild(repo, observed)
				if err != nil {
					resultsCh <- multiDryRunTaskResult{index: i, err: err}
					return
				}
			}
			result, err := repositoryPreflightExecute(repo, observed, single, *audit)
			resultsCh <- multiDryRunTaskResult{index: i, result: result, err: err}
		}()
	}
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	ordered := make([]apply.Result, len(spec.Repos))
	orderedErrors := make([]error, len(spec.Repos))
	for item := range resultsCh {
		if item.err != nil {
			item.result.ID = spec.Repos[item.index].ID
			item.result.UID = spec.Repos[item.index].DurableID()
			item.result.DryRun = true
			item.result.Error = item.err.Error()
		}
		ordered[item.index] = item.result
		orderedErrors[item.index] = item.err
	}

	output := apply.Output{Kind: apply.OutputKind, Version: apply.OutputVersion, Results: ordered}
	printApply(output)
	failed := 0
	var firstErr error
	for _, itemErr := range orderedErrors {
		if itemErr == nil {
			continue
		}
		failed++
		if firstErr == nil {
			firstErr = itemErr
		}
	}
	if failed > 0 {
		fmt.Fprintf(os.Stderr, "repoctl: %d repositories failed during audited dry-run: %v\n", failed, firstErr)
		return 1
	}
	return 0
}

func oneRepositoryArtifact(artifact planartifact.Artifact, index int) planartifact.Artifact {
	return planartifact.Artifact{
		Version:      artifact.Version,
		Kind:         artifact.Kind,
		Repositories: []planartifact.Repository{artifact.Repositories[index]},
	}
}
