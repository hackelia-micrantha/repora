package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sync"

	"repoctl/internal/apply"
	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/plan"
	"repoctl/internal/status"
)

type jsonOutput struct {
	Repos []jsonRepo `json:"repos"`
}

type jsonRepo struct {
	ID        string       `json:"id"`
	Canonical jsonRef      `json:"canonical"`
	Mirrors   []jsonMirror `json:"mirrors"`
}

type jsonRef struct {
	Ref    string `json:"ref"`
	Commit string `json:"commit"`
}

type jsonMirror struct {
	Provider string       `json:"provider"`
	Ref      string       `json:"ref"`
	Commit   string       `json:"commit"`
	State    status.State `json:"state"`
	Ahead    int          `json:"ahead"`
	Behind   int          `json:"behind"`
}

type repoResult struct {
	index  int
	result status.Result
	err    error
}

type checkSummary struct {
	results     []status.Result
	ok          []bool
	firstErr    error
	failedCount int
	failureCode int
}

var statusCheck = func(repo config.Repo) (status.Result, error) {
	return status.Check(repo, gitwrap.Client{})
}

var applyRepo = func(repo config.Repo, result status.Result, force bool) (apply.RepoApply, error) {
	return apply.Execute(repo, result, gitwrap.Client{}, force)
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: repoctl <status|plan|apply|sync> -f repora.yaml [--json] [--parallel N] [--continue-on-error]")
		return 1
	}

	command := args[0]
	if command != "status" && command != "plan" && command != "apply" && command != "sync" {
		fmt.Fprintln(os.Stderr, "usage: repoctl <status|plan|apply|sync> -f repora.yaml [--json] [--parallel N] [--continue-on-error]")
		return 1
	}

	flags := flag.NewFlagSet("repoctl "+command, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("f", "repora.yaml", "path to SCHEMA-0001 YAML config")
	jsonFlag := flags.Bool("json", false, "print JSON")
	parallelFlag := flags.Int("parallel", 5, "max number of concurrent repository checks")
	continueOnError := flags.Bool("continue-on-error", false, "continue processing repos after an error")
	force := flags.Bool("force", false, "overwrite mirror refs from canonical when mirror is ahead or diverged")

	if err := flags.Parse(args[1:]); err != nil {
		return 1
	}

	spec, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: %v\n", err)
		return 1
	}

	parallel := *parallelFlag
	if parallel < 1 {
		parallel = 1
	}

	summary := checkRepos(spec, parallel)
	if summary.firstErr != nil && !*continueOnError {
		fmt.Fprintf(os.Stderr, "repoctl: %v\n", summary.firstErr)
		return 1
	}

	switch command {
	case "status":
		return runStatus(spec, summary, *jsonFlag)
	case "plan":
		return runPlan(spec, summary, *jsonFlag)
	case "apply", "sync":
		return runApply(spec, summary, *jsonFlag, *force)
	default:
		panic("unreachable command validation")
	}
}

func checkRepos(spec config.Spec, parallel int) checkSummary {
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

			result, err := statusCheck(repo)
			resultsCh <- repoResult{index: i, result: result, err: err}
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
		if rr.err != nil {
			summary.failedCount++
			if summary.firstErr == nil {
				summary.firstErr = rr.err
			}
			continue
		}
		summary.results[rr.index] = rr.result
		summary.ok[rr.index] = true
		if rr.result.State == status.StateAhead || rr.result.State == status.StateDiverged {
			summary.failureCode = 2
		}
	}
	return summary
}

func runStatus(spec config.Spec, summary checkSummary, jsonFlag bool) int {
	orderedResults := make([]status.Result, 0, len(spec.Repos)-summary.failedCount)
	for i, result := range summary.results {
		if summary.ok[i] {
			orderedResults = append(orderedResults, result)
		}
	}

	if jsonFlag {
		if err := json.NewEncoder(os.Stdout).Encode(newJSONOutput(spec, summary.results, summary.ok)); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write json: %v\n", err)
			return 1
		}
	} else {
		for _, result := range orderedResults {
			printHuman(result)
		}
	}

	if summary.firstErr != nil {
		fmt.Fprintf(os.Stderr, "repoctl: %d repos failed; continuing due to --continue-on-error\n", summary.failedCount)
		if summary.failureCode == 2 {
			return 2
		}
		return 1
	}

	return summary.failureCode
}

func runPlan(spec config.Spec, summary checkSummary, jsonFlag bool) int {
	planOutput := plan.NewOutput(spec, summary.results, summary.ok)
	if jsonFlag {
		if err := json.NewEncoder(os.Stdout).Encode(planOutput); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write json: %v\n", err)
			return 1
		}
	} else {
		printPlan(planOutput)
	}

	if summary.firstErr != nil {
		fmt.Fprintf(os.Stderr, "repoctl: %d repos failed; continuing due to --continue-on-error\n", summary.failedCount)
		if summary.failureCode == 2 {
			return 2
		}
		return 1
	}

	return summary.failureCode
}

func runApply(spec config.Spec, summary checkSummary, jsonFlag bool, force bool) int {
	if summary.firstErr != nil && !force {
		fmt.Fprintf(os.Stderr, "repoctl: %d repos failed; refusing apply\n", summary.failedCount)
		return 1
	}
	if !force {
		for i, result := range summary.results {
			if summary.ok[i] && apply.IsUnsafe(result) {
				fmt.Fprintf(os.Stderr, "repoctl: repo %q mirror state is %s; rerun apply with --force to overwrite mirror from canonical\n", spec.Repos[i].ID, result.State)
				return 2
			}
		}
	}

	output := apply.Output{Apply: make([]apply.RepoApply, 0, len(spec.Repos)-summary.failedCount)}
	for i, repo := range spec.Repos {
		if !summary.ok[i] {
			continue
		}
		repoApply, err := applyRepo(repo, summary.results[i], force)
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: %v\n", err)
			return 1
		}
		output.Apply = append(output.Apply, repoApply)
	}

	if jsonFlag {
		if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write json: %v\n", err)
			return 1
		}
	} else {
		printApply(output)
	}
	return 0
}

func newJSONOutput(spec config.Spec, results []status.Result, ok []bool) jsonOutput {
	out := jsonOutput{Repos: make([]jsonRepo, 0, len(spec.Repos))}
	for i, repo := range spec.Repos {
		if !ok[i] {
			continue
		}
		result := results[i]
		out.Repos = append(out.Repos, jsonRepo{
			ID: repo.ID,
			Canonical: jsonRef{
				Ref:    "HEAD",
				Commit: result.Canonical,
			},
			Mirrors: []jsonMirror{
				{
					Provider: repo.Mirrors[0].Provider,
					Ref:      "HEAD",
					Commit:   result.Mirror,
					State:    result.State,
					Ahead:    result.Ahead,
					Behind:   result.Behind,
				},
			},
		})
	}
	return out
}

func printHuman(result status.Result) {
	fmt.Println(result.ID)
	if result.Canonical != "" {
		fmt.Printf("  canonical: %s\n", result.Canonical)
	}
	if result.Mirror != "" {
		fmt.Printf("  mirror:    %s\n", result.Mirror)
	}
	switch result.State {
	case status.StateBehind:
		fmt.Printf("  state:     %s (%d)\n", result.State, result.Behind)
	case status.StateAhead:
		fmt.Printf("  state:     %s (%d)\n", result.State, result.Ahead)
	case status.StateDiverged:
		fmt.Printf("  state:     %s (behind %d, ahead %d)\n", result.State, result.Behind, result.Ahead)
	default:
		fmt.Printf("  state:     %s\n", result.State)
	}
}

func printPlan(output plan.Output) {
	for _, repoPlan := range output.Plan {
		fmt.Println(repoPlan.ID)
		if len(repoPlan.Actions) == 0 {
			fmt.Println("  no changes")
			continue
		}
		for _, action := range repoPlan.Actions {
			fmt.Printf("  push mirror %s: %d commits\n", action.Target, action.Behind)
		}
	}
}

func printApply(output apply.Output) {
	for _, repoApply := range output.Apply {
		fmt.Println(repoApply.ID)
		if len(repoApply.Actions) == 0 {
			fmt.Println("  no changes")
			continue
		}
		for _, action := range repoApply.Actions {
			if action.Destructive {
				fmt.Printf("  force push mirror %s\n", action.Target)
				continue
			}
			fmt.Printf("  push mirror %s\n", action.Target)
		}
	}
}
