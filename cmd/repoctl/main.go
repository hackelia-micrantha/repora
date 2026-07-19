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
	UID       string       `json:"uid"`
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
	terr   error
}

type applyTaskResult struct {
	index  int
	result apply.Result
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

var applyExecute = func(repo config.Repo, result status.Result, force, dryRun bool) (apply.Result, error) {
	return apply.Execute(repo, result, gitwrap.Client{}, force, dryRun)
}

var progressf = func(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
}

var debugf = func(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if isHelpRequest(args) {
		printHelp(os.Stdout)
		return 0
	}

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: repoctl <status|plan|apply|sync> -f repora.yaml [--json] [--parallel N] [--continue-on-error] [--dry-run] [--force] [--debug]")
		return 1
	}

	command := args[0]
	if command != "status" && command != "plan" && command != "apply" && command != "sync" {
		fmt.Fprintln(os.Stderr, "usage: repoctl <status|plan|apply|sync> -f repora.yaml [--json] [--parallel N] [--continue-on-error] [--dry-run] [--force] [--debug]")
		return 1
	}

	flags := flag.NewFlagSet("repoctl "+command, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("f", "repora.yaml", "path to SCHEMA-0001 YAML config")
	jsonFlag := flags.Bool("json", false, "print JSON")
	parallelFlag := flags.Int("parallel", 5, "max number of concurrent repository operations")
	continueOnError := flags.Bool("continue-on-error", false, "continue processing repos after an error")
	dryRun := flags.Bool("dry-run", false, "show what would change without mutating mirror state")
	force := flags.Bool("force", false, "allow destructive mirror overwrites for ahead or diverged mirrors")
	debug := flags.Bool("debug", false, "print debug logs to stderr")

	if err := flags.Parse(args[1:]); err != nil {
		return 1
	}

	if *dryRun && command != "status" && command != "plan" && command != "apply" && command != "sync" {
		fmt.Fprintln(os.Stderr, "repoctl: --dry-run is only supported for status, plan, apply, and sync")
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

	if *debug {
		debugf("repoctl: debug command=%s repos=%d parallel=%d dry_run=%t\n", command, len(spec.Repos), parallel, *dryRun)
	}

	summary := checkRepos(spec, parallel, *debug)
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
		return runApply(spec, summary, *jsonFlag, *force, *dryRun, parallel)
	default:
		panic("unreachable command validation")
	}
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

func runApply(spec config.Spec, summary checkSummary, jsonFlag bool, force bool, dryRun bool, parallel int) int {
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
	output, err := applyRepos(spec, summary, force, dryRun, parallel, !jsonFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: %v\n", err)
		return 1
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

func applyRepos(spec config.Spec, summary checkSummary, force bool, dryRun bool, parallel int, progress bool) (apply.Output, error) {
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
			result, err := applyExecute(repo, summary.results[i], force, dryRun)
			resultsCh <- applyTaskResult{index: i, result: result, err: err}
		}()
	}
	go func() {
		wg.Wait()
		close(resultsCh)
	}()
	ordered := make([]apply.Result, len(spec.Repos))
	ok := make([]bool, len(spec.Repos))
	for res := range resultsCh {
		if res.err != nil {
			res.result.Error = res.err.Error()
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
	return output, nil
}

func newJSONOutput(spec config.Spec, results []status.Result, ok []bool) jsonOutput {
	out := jsonOutput{Repos: make([]jsonRepo, 0, len(spec.Repos))}
	for i, repo := range spec.Repos {
		if !ok[i] {
			continue
		}
		result := results[i]
		out.Repos = append(out.Repos, jsonRepo{
			ID:  repo.ID,
			UID: repo.DurableID(),
			Canonical: jsonRef{Ref: "HEAD", Commit: result.Canonical},
			Mirrors: []jsonMirror{{
				Provider: repo.Mirrors[0].Provider,
				Ref:      "HEAD",
				Commit:   result.Mirror,
				State:    result.State,
				Ahead:    result.Ahead,
				Behind:   result.Behind,
			}},
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
	for _, result := range output.Results {
		fmt.Println(result.ID)
		if len(result.Actions) == 0 {
			fmt.Println("  no changes")
			continue
		}
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
		if result.Error != "" {
			fmt.Printf("  error: %s\n", result.Error)
		}
	}
}
