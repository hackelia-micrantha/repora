package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/status"
)

type jsonOutput struct {
	Repos []status.Result `json:"repos"`
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	flags := flag.NewFlagSet("repoctl status", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("f", "repora.yaml", "path to SCHEMA-0001 YAML config")
	jsonFlag := flags.Bool("json", false, "print JSON")

	if len(args) == 0 || args[0] != "status" {
		fmt.Fprintln(os.Stderr, "usage: repoctl status -f repora.yaml [--json]")
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

	results := make([]status.Result, 0, len(spec.Repos))
	code := 0
	for _, repo := range spec.Repos {
		result, err := status.Check(repo, gitwrap.Client{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: %v\n", err)
			return 1
		}
		results = append(results, result)
		if result.State == status.StateAhead || result.State == status.StateDiverged {
			code = 2
		}
	}

	if *jsonFlag {
		if err := json.NewEncoder(os.Stdout).Encode(jsonOutput{Repos: results}); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write json: %v\n", err)
			return 1
		}
	} else {
		for _, result := range results {
			printHuman(result)
		}
	}

	return code
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
