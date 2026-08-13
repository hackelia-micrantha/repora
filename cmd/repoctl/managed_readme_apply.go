package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"repoctl/internal/config"
	"repoctl/internal/managedartifact"
)

var managedREADMEPreflight = managedartifact.PreflightPlan
var managedREADMEPreflightObserver = func() managedartifact.READMEObserver { return managedartifact.NewGitREADMEObserver() }

func runApplyREADME(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(os.Stdout, "usage: repoctl apply-readme -f repora.yaml --plan-file FILE --dry-run")
		return 0
	}

	flags := flag.NewFlagSet("repoctl apply-readme", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("f", "repora.yaml", "path to SCHEMA-0001 YAML config")
	planFile := flags.String("plan-file", "", "exact repora.io/managed-artifact-plan v1 JSON to preflight")
	dryRun := flags.Bool("dry-run", false, "validate the exact plan and show its reviewed diff without mutation")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "repoctl: apply-readme does not accept positional arguments")
		return 1
	}
	if !*dryRun {
		fmt.Fprintln(os.Stderr, "repoctl: managed README apply is not implemented; apply-readme currently requires --dry-run")
		return 1
	}
	if *planFile == "" {
		fmt.Fprintln(os.Stderr, "repoctl: apply-readme --dry-run requires --plan-file FILE")
		return 1
	}

	spec, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: %v\n", err)
		return 1
	}
	data, err := os.ReadFile(*planFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: read managed README plan: %v\n", err)
		return 1
	}
	plan, err := managedartifact.ParsePlan(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: %v\n", err)
		return 1
	}
	if err := managedREADMEPreflight(spec, plan, managedREADMEPreflightObserver()); err != nil {
		if errors.Is(err, managedartifact.ErrStale) {
			fmt.Fprintf(os.Stderr, "repoctl: %v\n", err)
			return 2
		}
		fmt.Fprintf(os.Stderr, "repoctl: managed README dry-run preflight: %v\n", err)
		return 1
	}
	printManagedREADMEPlan(os.Stdout, plan)
	return 0
}
