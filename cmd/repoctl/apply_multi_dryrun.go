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

type preparedRepository struct {
	index    int
	observed status.RepositoryResult
	artifact planartifact.Artifact
	err      error
}

type detailedTaskResult struct {
	index  int
	result apply.DetailedResult
	err    error
}

var repositoryArtifactExecute = func(repo config.Repo, observed status.RepositoryResult, artifact planartifact.Artifact, allowForce, dryRun bool, audit apply.Audit) (apply.DetailedResult, error) {
	return apply.ExecuteRepositoryArtifactAudited(repo, observed, artifact, gitwrap.Client{}, allowForce, dryRun, audit)
}

func runPathBoundApply(spec config.Spec, parallel int, jsonFlag, force, dryRun bool, artifact *planartifact.Artifact, configPath string, debug bool) int {
	if artifact != nil && len(artifact.Repositories) != len(spec.Repos) {
		fmt.Fprintf(os.Stderr, "repoctl: plan artifact contains %d repositories for %d selected configuration repositories\n", len(artifact.Repositories), len(spec.Repos))
		return 1
	}

	prepared := preparePathBoundRepositories(spec, parallel, artifact, debug)
	if code := renderPreparationFailures(spec, prepared, jsonFlag, dryRun); code != 0 {
		return code
	}

	combined := planartifact.Artifact{
		Version:      planartifact.Version,
		Kind:         planartifact.Kind,
		Repositories: make([]planartifact.Repository, len(prepared)),
	}
	for i, item := range prepared {
		combined.Repositories[i] = item.artifact.Repositories[0]
	}
	if !dryRun && !force {
		requiresForce, err := apply.ArtifactRequiresForce(combined)
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: validate exact plan artifact: %v\n", err)
			return 1
		}
		if requiresForce {
			fmt.Fprintln(os.Stderr, "repoctl: exact plan contains forced mirror actions; rerun with --force")
			return 2
		}
	}

	audit, err := newAudit(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: initialize execution journal: %v\n", err)
		return 1
	}

	sem := make(chan struct{}, parallel)
	resultsCh := make(chan detailedTaskResult, len(spec.Repos))
	var wg sync.WaitGroup
	wg.Add(len(spec.Repos))
	for i, repo := range spec.Repos {
		i := i
		repo := repo
		item := prepared[i]
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if debug {
				debugf("repoctl: debug executing repo=%s mirrors=%d dry_run=%t\n", repo.ID, len(repo.Mirrors), dryRun)
			}
			result, err := repositoryArtifactExecute(repo, item.observed, item.artifact, force, dryRun, *audit)
			resultsCh <- detailedTaskResult{index: i, result: result, err: err}
		}()
	}
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	ordered := make([]apply.DetailedResult, len(spec.Repos))
	orderedErrors := make([]error, len(spec.Repos))
	for item := range resultsCh {
		if item.err != nil {
			item.result.Error = item.err.Error()
		}
		ordered[item.index] = item.result
		orderedErrors[item.index] = item.err
	}
	output := apply.NewDetailedOutput(ordered)
	if jsonFlag {
		if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write json: %v\n", err)
			return 1
		}
	} else {
		printDetailedApply(output)
	}

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
		fmt.Fprintf(os.Stderr, "repoctl: %d repositories failed during apply: %v\n", failed, firstErr)
		return 1
	}
	return 0
}

func preparePathBoundRepositories(spec config.Spec, parallel int, artifact *planartifact.Artifact, debug bool) []preparedRepository {
	sem := make(chan struct{}, parallel)
	resultsCh := make(chan preparedRepository, len(spec.Repos))
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
				debugf("repoctl: debug preparing repo=%s mirrors=%d\n", repo.ID, len(repo.Mirrors))
			}
			observed, err := statusCheckAll(repo)
			if err != nil {
				resultsCh <- preparedRepository{index: i, observed: observed, err: err}
				return
			}
			var single planartifact.Artifact
			if artifact != nil {
				single = oneRepositoryArtifact(*artifact, i)
			} else {
				single, err = repositoryPlanBuild(repo, observed)
			}
			resultsCh <- preparedRepository{index: i, observed: observed, artifact: single, err: err}
		}()
	}
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	ordered := make([]preparedRepository, len(spec.Repos))
	for item := range resultsCh {
		ordered[item.index] = item
	}
	return ordered
}

func renderPreparationFailures(spec config.Spec, prepared []preparedRepository, jsonFlag, dryRun bool) int {
	results := make([]apply.DetailedResult, len(spec.Repos))
	failed := 0
	var firstErr error
	for i, item := range prepared {
		if item.err == nil {
			continue
		}
		failed++
		if firstErr == nil {
			firstErr = item.err
		}
		results[i] = apply.DetailedResult{
			ID:      spec.Repos[i].ID,
			UID:     spec.Repos[i].DurableID(),
			DryRun:  dryRun,
			Actions: []apply.DetailedAction{},
			Error:   item.err.Error(),
		}
	}
	if failed == 0 {
		return 0
	}
	output := apply.NewDetailedOutput(results)
	if jsonFlag {
		if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write json: %v\n", err)
			return 1
		}
	} else {
		printDetailedApply(output)
	}
	fmt.Fprintf(os.Stderr, "repoctl: %d repositories failed before execution: %v\n", failed, firstErr)
	return 1
}

func printDetailedApply(output apply.DetailedOutput) {
	for _, result := range output.Results {
		fmt.Println(result.ID)
		if len(result.Actions) == 0 {
			fmt.Println("  no changes")
		}
		for _, action := range result.Actions {
			mode := "apply"
			if result.DryRun {
				mode = "dry-run"
			}
			force := ""
			if action.Force {
				force = " (force)"
			}
			fmt.Printf("  %s %s %s -> %s: %s%s\n", mode, action.Type, action.Source, action.Target, action.Outcome, force)
			if action.Error != "" {
				fmt.Printf("    error: %s\n", action.Error)
			}
		}
		if result.Journal != nil {
			if result.Journal.ExecutionID != "" {
				fmt.Printf("  execution: %s\n", result.Journal.ExecutionID)
			}
			if result.Journal.Intent != "" {
				fmt.Printf("  journal intent: %s\n", result.Journal.Intent)
			}
			if result.Journal.Result != "" {
				fmt.Printf("  journal result: %s\n", result.Journal.Result)
			}
		}
		if result.Error != "" {
			fmt.Printf("  error: %s\n", result.Error)
		}
	}
}

func oneRepositoryArtifact(artifact planartifact.Artifact, index int) planartifact.Artifact {
	return planartifact.Artifact{
		Version:      artifact.Version,
		Kind:         artifact.Kind,
		Repositories: []planartifact.Repository{artifact.Repositories[index]},
	}
}
