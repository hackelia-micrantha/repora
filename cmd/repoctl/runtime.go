package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"repoctl/internal/apply"
	"repoctl/internal/config"
	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

func selectArtifactSpec(spec config.Spec, artifact planartifact.Artifact) (config.Spec, error) {
	if len(artifact.Repositories) == 0 {
		return config.Spec{}, fmt.Errorf("plan artifact requires at least one repository")
	}
	configured := make(map[string]config.Repo, len(spec.Repos))
	for _, repo := range spec.Repos {
		configured[repo.DurableID()] = repo
	}
	selected := config.Spec{Repos: make([]config.Repo, 0, len(artifact.Repositories))}
	seen := make(map[string]struct{}, len(artifact.Repositories))
	for i, planned := range artifact.Repositories {
		if _, exists := seen[planned.UID]; exists {
			return config.Spec{}, fmt.Errorf("plan artifact repository %d duplicates uid %q", i, planned.UID)
		}
		seen[planned.UID] = struct{}{}
		repo, ok := configured[planned.UID]
		if !ok {
			return config.Spec{}, fmt.Errorf("plan artifact repository uid %q is not present in configuration", planned.UID)
		}
		selected.Repos = append(selected.Repos, repo)
	}
	return selected, nil
}

func checkRepos(spec config.Spec, parallel int, debug bool) checkSummary {
	sem := make(chan struct{}, parallel)
	resultsCh := make(chan repoResult, len(spec.Repos))
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
			result, err := statusCheck(repo)
			resultsCh <- repoResult{index: i, result: result, terr: err}
		}()
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	summary := checkSummary{
		results: make([]status.Result, len(spec.Repos)),
		ok:      make([]bool, len(spec.Repos)),
	}
	for rr := range resultsCh {
		if rr.terr != nil {
			summary.failedCount++
			if summary.firstErr == nil {
				summary.firstErr = rr.terr
			}
			continue
		}
		summary.results[rr.index] = rr.result
		summary.ok[rr.index] = true
		if debug {
			debugf("repoctl: debug checked repo=%s state=%s\n", spec.Repos[rr.index].ID, rr.result.State)
		}
		if rr.result.State == status.StateAhead || rr.result.State == status.StateDiverged {
			summary.failureCode = 2
		}
	}
	return summary
}

func runPlan(spec config.Spec, summary checkSummary, jsonFlag bool, artifactFlag bool, force bool) int {
	artifact, plannedResults, buildFailures, buildErr := buildPlanArtifact(spec, summary)
	if artifactFlag && (summary.firstErr != nil || buildErr != nil) {
		fmt.Fprintln(os.Stderr, "repoctl: exact plan artifact not emitted because planning was incomplete")
		if buildErr != nil {
			fmt.Fprintf(os.Stderr, "repoctl: %d repositories failed during planning: %v\n", buildFailures, buildErr)
		}
		return 1
	}

	plans, err := artifact.Plans()
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: validate generated plan artifact: %v\n", err)
		return 1
	}
	planOutput := plan.NewOutput(plans, plannedResults)
	if artifactFlag {
		if err := json.NewEncoder(os.Stdout).Encode(artifact); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write plan artifact: %v\n", err)
			return 1
		}
	} else if jsonFlag {
		if err := json.NewEncoder(os.Stdout).Encode(planOutput); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write json: %v\n", err)
			return 1
		}
	} else {
		printPlan(planOutput)
	}

	if buildErr != nil {
		fmt.Fprintf(os.Stderr, "repoctl: %d repositories failed during planning: %v\n", buildFailures, buildErr)
		return 1
	}
	if summary.firstErr != nil {
		fmt.Fprintf(os.Stderr, "repoctl: %d repos failed; continuing due to --continue-on-error\n", summary.failedCount)
		if summary.failureCode == 2 {
			return 2
		}
		return 1
	}
	if !force && summary.failureCode == 2 {
		return 2
	}
	return 0
}

func buildPlanArtifact(spec config.Spec, summary checkSummary) (planartifact.Artifact, []status.Result, int, error) {
	artifact := planartifact.Artifact{
		Version:      planartifact.Version,
		Kind:         planartifact.Kind,
		Repositories: []planartifact.Repository{},
	}
	results := make([]status.Result, 0, len(spec.Repos)-summary.failedCount)
	var firstErr error
	failedCount := 0
	for i, repo := range spec.Repos {
		if !summary.ok[i] {
			continue
		}
		single, err := planBuild(repo, summary.results[i], true)
		if err != nil {
			failedCount++
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if len(single.Repositories) != 1 {
			failedCount++
			err := fmt.Errorf("repo %q planner returned %d repositories", repo.ID, len(single.Repositories))
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		artifact.Repositories = append(artifact.Repositories, single.Repositories[0])
		results = append(results, summary.results[i])
	}
	if err := artifact.Validate(); err != nil {
		return planartifact.Artifact{}, nil, failedCount + 1, fmt.Errorf("validate generated artifact: %w", err)
	}
	return artifact, results, failedCount, firstErr
}

func runApply(spec config.Spec, summary checkSummary, jsonFlag bool, force bool, dryRun bool, parallel int, artifact *planartifact.Artifact, configPath string) int {
	var output apply.Output
	var applyErr error

	if artifact != nil {
		if summary.firstErr != nil {
			fmt.Fprintf(os.Stderr, "repoctl: %d repos failed; refusing exact plan apply\n", summary.failedCount)
			return 1
		}
		requiresForce, err := apply.ArtifactRequiresForce(*artifact)
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: validate plan artifact: %v\n", err)
			return 1
		}
		if requiresForce && !force && !dryRun {
			fmt.Fprintln(os.Stderr, "repoctl: plan artifact contains a forced action; rerun apply with --force")
			return 2
		}
	} else {
		if summary.firstErr != nil && !force {
			fmt.Fprintf(os.Stderr, "repoctl: %d repos failed; refusing apply\n", summary.failedCount)
			return 1
		}
		if !force && !dryRun {
			for i, result := range summary.results {
				if summary.ok[i] && apply.IsUnsafe(result) {
					fmt.Fprintf(os.Stderr, "repoctl: repo %q mirror state is %s; rerun apply with --force to overwrite mirror from canonical\n", spec.Repos[i].ID, result.State)
					return 2
				}
			}
		}
	}

	audit, err := newAudit(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: initialize execution journal: %v\n", err)
		return 1
	}
	if artifact != nil {
		output, applyErr = applyArtifactRepos(spec, summary, *artifact, force, dryRun, parallel, !jsonFlag, audit)
	} else {
		output, applyErr = applyRepos(spec, summary, force, dryRun, parallel, !jsonFlag, audit)
	}

	if jsonFlag {
		if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write json: %v\n", err)
			return 1
		}
	} else {
		printApply(output)
	}
	if applyErr != nil {
		fmt.Fprintf(os.Stderr, "repoctl: %v\n", applyErr)
		return 1
	}
	return 0
}

func applyRepos(spec config.Spec, summary checkSummary, force bool, dryRun bool, parallel int, progress bool, audit *apply.Audit) (apply.Output, error) {
	return collectApplyResults(spec, summary, parallel, progress, func(i int, repo config.Repo) (apply.Result, error) {
		if audit == nil {
			return applyExecute(repo, summary.results[i], force, dryRun)
		}
		return auditedApplyExecute(repo, summary.results[i], force, dryRun, *audit)
	})
}

func applyArtifactRepos(spec config.Spec, summary checkSummary, artifact planartifact.Artifact, force bool, dryRun bool, parallel int, progress bool, audit *apply.Audit) (apply.Output, error) {
	if len(artifact.Repositories) != len(spec.Repos) {
		return apply.Output{}, fmt.Errorf("plan artifact contains %d repositories for %d selected configuration repositories", len(artifact.Repositories), len(spec.Repos))
	}
	return collectApplyResults(spec, summary, parallel, progress, func(i int, repo config.Repo) (apply.Result, error) {
		single := planartifact.Artifact{
			Version:      artifact.Version,
			Kind:         artifact.Kind,
			Repositories: []planartifact.Repository{artifact.Repositories[i]},
		}
		if audit == nil {
			return artifactApplyExecute(repo, summary.results[i], single, force, dryRun)
		}
		return auditedArtifactApplyExecute(repo, summary.results[i], single, force, dryRun, *audit)
	})
}

func collectApplyResults(spec config.Spec, summary checkSummary, parallel int, progress bool, execute func(int, config.Repo) (apply.Result, error)) (apply.Output, error) {
	sem := make(chan struct{}, parallel)
	resultsCh := make(chan applyTaskResult, len(spec.Repos)-summary.failedCount)
	var wg sync.WaitGroup
	for i, repo := range spec.Repos {
		if !summary.ok[i] {
			continue
		}
		i := i
		repo := repo
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if progress {
				progressf("repoctl: applying %s\n", repo.ID)
			}
			result, err := execute(i, repo)
			resultsCh <- applyTaskResult{index: i, result: result, err: err}
		}()
	}
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	ordered := make([]apply.Result, len(spec.Repos))
	orderedErrors := make([]error, len(spec.Repos))
	ok := make([]bool, len(spec.Repos))
	for res := range resultsCh {
		if res.err != nil {
			res.result.Error = res.err.Error()
			orderedErrors[res.index] = res.err
		}
		ordered[res.index] = res.result
		ok[res.index] = true
	}
	output := apply.Output{Results: make([]apply.Result, 0, len(spec.Repos)-summary.failedCount)}
	for i, res := range ordered {
		if ok[i] {
			output.Results = append(output.Results, res)
		}
	}
	var firstErr error
	failedCount := 0
	for _, err := range orderedErrors {
		if err == nil {
			continue
		}
		failedCount++
		if firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		noun := "repositories"
		if failedCount == 1 {
			noun = "repository"
		}
		return output, fmt.Errorf("%d %s failed during apply: %w", failedCount, noun, firstErr)
	}
	return output, nil
}

func printPlan(output plan.Output) {
	for _, repoPlan := range output.Plan {
		fmt.Println(repoPlan.ID)
		if len(repoPlan.Actions) == 0 {
			fmt.Println("  no changes")
			continue
		}
		for _, action := range repoPlan.Actions {
			if action.Destructive {
				fmt.Printf("  overwrite mirror %s (destructive)\n", action.Target)
				continue
			}
			fmt.Printf("  push mirror %s: %d commits\n", action.Target, action.Behind)
		}
	}
}

func printApply(output apply.Output) {
	for _, result := range output.Results {
		fmt.Println(result.ID)
		if len(result.Actions) == 0 {
			fmt.Println("  no changes")
		} else {
			for _, action := range result.Actions {
				mode := "apply"
				if result.DryRun {
					mode = "dry-run"
				}
				if action.Force {
					fmt.Printf("  %s %s %s -> %s (force)\n", mode, action.Type, action.Source, action.Target)
				} else {
					fmt.Printf("  %s %s %s -> %s\n", mode, action.Type, action.Source, action.Target)
				}
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
