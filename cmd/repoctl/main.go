package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sync"

	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
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

var statusCheck = func(repo config.Repo) (status.Result, error) {
	return status.Check(repo, gitwrap.Client{})
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	flags := flag.NewFlagSet("repoctl status", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("f", "repora.yaml", "path to SCHEMA-0001 YAML config")
	jsonFlag := flags.Bool("json", false, "print JSON")
	parallelFlag := flags.Int("parallel", 5, "max number of concurrent repository checks")
	continueOnError := flags.Bool("continue-on-error", false, "continue processing repos after an error")

	if len(args) == 0 || args[0] != "status" {
		fmt.Fprintln(os.Stderr, "usage: repoctl status -f repora.yaml [--json] [--parallel N] [--continue-on-error]")
		return 1
	}
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

	results := make([]status.Result, len(spec.Repos))
	ok := make([]bool, len(spec.Repos))
	var firstErr error
	failureCode := 0
	failedCount := 0

	for rr := range resultsCh {
		if rr.err != nil {
			failedCount++
			if firstErr == nil {
				firstErr = rr.err
			}
			continue
		}
		results[rr.index] = rr.result
		ok[rr.index] = true
		if rr.result.State == status.StateAhead || rr.result.State == status.StateDiverged {
			failureCode = 2
		}
	}

	if firstErr != nil && !*continueOnError {
		fmt.Fprintf(os.Stderr, "repoctl: %v\n", firstErr)
		return 1
	}

	orderedResults := make([]status.Result, 0, len(spec.Repos)-failedCount)
	for i, result := range results {
		if ok[i] {
			orderedResults = append(orderedResults, result)
		}
	}

	if *jsonFlag {
		if err := json.NewEncoder(os.Stdout).Encode(newJSONOutput(spec, results, ok)); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write json: %v\n", err)
			return 1
		}
	} else {
		for _, result := range orderedResults {
			printHuman(result)
		}
	}

	if firstErr != nil {
		fmt.Fprintf(os.Stderr, "repoctl: %d repos failed; continuing due to --continue-on-error\n", failedCount)
		if failureCode == 2 {
			return 2
		}
		return 1
	}

	return failureCode
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
