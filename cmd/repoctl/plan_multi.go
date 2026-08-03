package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"repoctl/internal/apply"
	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

type repositoryPlanTaskResult struct {
	index    int
	artifact planartifact.Artifact
	observed status.RepositoryResult
	err      error
}

var repositoryPlanBuild = func(repo config.Repo, observed status.RepositoryResult) (planartifact.Artifact, error) {
	return apply.BuildRepositoryArtifact(repo, observed, gitwrap.Client{})
}

func hasMultiMirrorRepository(spec config.Spec) bool {
	for _, repo := range spec.Repos {
		if len(repo.Mirrors) > 1 {
			return true
		}
	}
	return false
}

func runMultiMirrorPlan(spec config.Spec, parallel int, jsonFlag, artifactFlag, force, continueOnError, debug bool) int {
	if jsonFlag {
		fmt.Fprintln(os.Stderr, "repoctl: multi-mirror plan --json is not supported; use --artifact for the exact versioned plan")
		return 1
	}

	sem := make(chan struct{}, parallel)
	resultsCh := make(chan repositoryPlanTaskResult, len(spec.Repos))
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
				debugf("repoctl: debug planning repo=%s mirrors=%d\n", repo.ID, len(repo.Mirrors))
			}
			observed, err := statusCheckAll(repo)
			if err == nil {
				var artifact planartifact.Artifact
				artifact, err = repositoryPlanBuild(repo, observed)
				resultsCh <- repositoryPlanTaskResult{index: i, artifact: artifact, observed: observed, err: err}
				return
			}
			resultsCh <- repositoryPlanTaskResult{index: i, observed: observed, err: err}
		}()
	}
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	ordered := make([]repositoryPlanTaskResult, len(spec.Repos))
	for result := range resultsCh {
		ordered[result.index] = result
	}

	aggregate := planartifact.Artifact{
		Version:      planartifact.Version,
		Kind:         planartifact.Kind,
		Repositories: []planartifact.Repository{},
	}
	failedCount := 0
	var firstErr error
	unsafe := false
	for _, result := range ordered {
		if result.err != nil {
			failedCount++
			if firstErr == nil {
				firstErr = result.err
			}
			continue
		}
		if len(result.artifact.Repositories) != 1 {
			failedCount++
			if firstErr == nil {
				firstErr = fmt.Errorf("planner returned %d repositories, want 1", len(result.artifact.Repositories))
			}
			continue
		}
		aggregate.Repositories = append(aggregate.Repositories, result.artifact.Repositories[0])
		for _, mirror := range result.observed.Mirrors {
			if mirror.State == status.StateAhead || mirror.State == status.StateDiverged {
				unsafe = true
			}
		}
	}

	if failedCount > 0 && !continueOnError {
		if artifactFlag {
			fmt.Fprintln(os.Stderr, "repoctl: exact plan artifact not emitted because planning was incomplete")
		}
		fmt.Fprintf(os.Stderr, "repoctl: %d repositories failed during planning: %v\n", failedCount, firstErr)
		return 1
	}
	if err := aggregate.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: validate generated plan artifact: %v\n", err)
		return 1
	}

	if artifactFlag {
		if failedCount > 0 {
			fmt.Fprintln(os.Stderr, "repoctl: exact plan artifact not emitted because planning was incomplete")
			fmt.Fprintf(os.Stderr, "repoctl: %d repositories failed during planning: %v\n", failedCount, firstErr)
			return 1
		}
		if err := json.NewEncoder(os.Stdout).Encode(aggregate); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write plan artifact: %v\n", err)
			return 1
		}
	} else {
		printExactMultiMirrorPlan(aggregate)
	}

	if failedCount > 0 {
		fmt.Fprintf(os.Stderr, "repoctl: %d repositories failed during planning: %v\n", failedCount, firstErr)
		return 1
	}
	if unsafe && !force {
		return 2
	}
	return 0
}

func printExactMultiMirrorPlan(artifact planartifact.Artifact) {
	for _, repo := range artifact.Repositories {
		fmt.Println(repo.ID)
		if len(repo.Actions) == 0 {
			fmt.Println("  no changes")
			continue
		}
		for _, action := range repo.Actions {
			target := action.Target.Provider + ":" + action.Target.Path
			if action.Force {
				fmt.Printf("  overwrite mirror %s (destructive)\n", target)
			} else {
				fmt.Printf("  push mirror %s\n", target)
			}
		}
	}
}
