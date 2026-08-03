package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/status"
)

type statusTaskResult struct {
	index  int
	result status.RepositoryResult
	err    error
}

var statusCheckAll = func(repo config.Repo) (status.RepositoryResult, error) {
	if len(repo.Mirrors) == 1 {
		observed, err := statusCheck(repo)
		if err != nil {
			return status.RepositoryResult{}, err
		}
		return status.ProjectSingle(repo, observed)
	}
	return status.CheckAll(repo, gitwrap.Client{})
}

func runMultiStatus(spec config.Spec, parallel int, jsonFlag, continueOnError, debug bool) int {
	sem := make(chan struct{}, parallel)
	resultsCh := make(chan statusTaskResult, len(spec.Repos))
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
				debugf("repoctl: debug checking repo=%s\n", repo.ID)
			}
			result, err := statusCheckAll(repo)
			resultsCh <- statusTaskResult{index: i, result: result, err: err}
		}()
	}
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	ordered := make([]status.RepositoryResult, len(spec.Repos))
	orderedErrors := make([]error, len(spec.Repos))
	for item := range resultsCh {
		ordered[item.index] = item.result
		orderedErrors[item.index] = item.err
		if debug {
			debugf("repoctl: debug checked repo=%s mirrors=%d error=%t\n", spec.Repos[item.index].ID, len(item.result.Mirrors), item.err != nil)
		}
	}

	fatalBeforeOutput := false
	failedRepositories := 0
	failureCode := 0
	for i, result := range ordered {
		if orderedErrors[i] != nil {
			failedRepositories++
			failureCode = 1
			if len(result.Mirrors) == 0 && !continueOnError {
				fatalBeforeOutput = true
			}
			continue
		}
		for _, mirror := range result.Mirrors {
			if mirror.State == status.StateAhead || mirror.State == status.StateDiverged {
				if failureCode == 0 {
					failureCode = 2
				}
			}
		}
	}
	if fatalBeforeOutput {
		for _, err := range orderedErrors {
			if err != nil {
				fmt.Fprintf(os.Stderr, "repoctl: %v\n", err)
				break
			}
		}
		return 1
	}

	output := status.Output{Kind: status.OutputKind, Version: status.OutputVersion, Repos: ordered}
	if jsonFlag {
		if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write json: %v\n", err)
			return 1
		}
	} else {
		for _, result := range ordered {
			printMultiStatus(result)
		}
	}
	if failedRepositories > 0 {
		fmt.Fprintf(os.Stderr, "repoctl: %d repositories have incomplete status evidence\n", failedRepositories)
	}
	return failureCode
}

func printMultiStatus(result status.RepositoryResult) {
	fmt.Println(result.ID)
	if result.Canonical.Commit != "" {
		fmt.Printf("  canonical: %s\n", result.Canonical.Commit)
	}
	if result.Error != "" {
		fmt.Printf("  error: %s\n", result.Error)
	}
	for _, mirror := range result.Mirrors {
		fmt.Printf("  mirror %s\n", mirror.Target)
		if mirror.Commit != "" {
			fmt.Printf("    commit: %s\n", mirror.Commit)
		}
		switch mirror.State {
		case status.StateBehind:
			fmt.Printf("    state:  %s (%d behind)\n", mirror.State, mirror.Behind)
		case status.StateAhead:
			fmt.Printf("    state:  %s (%d ahead)\n", mirror.State, mirror.Ahead)
		case status.StateDiverged:
			fmt.Printf("    state:  %s (%d behind, %d ahead)\n", mirror.State, mirror.Behind, mirror.Ahead)
		default:
			fmt.Printf("    state:  %s\n", mirror.State)
		}
		if mirror.Error != "" {
			fmt.Printf("    error:  %s\n", mirror.Error)
		}
	}
}

func requireSingleMirrorReconciliation(spec config.Spec) error {
	for _, repo := range spec.Repos {
		if len(repo.Mirrors) != 1 {
			return fmt.Errorf("repo %q configures %d mirrors; plan, apply, and sync remain single-mirror until multi-mirror execution is implemented", repo.ID, len(repo.Mirrors))
		}
	}
	return nil
}
