package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"repoctl/internal/config"
	"repoctl/internal/managedartifact"
	"repoctl/internal/managedartifactapply"
)

var managedREADMEPreflight = managedartifact.PreflightPlan
var managedREADMEPreflightObserver = func() managedartifact.READMEObserver { return managedartifact.NewGitREADMEObserver() }
var managedREADMEApplyExecute = managedartifactapply.Execute
var managedREADMECommitPreparer = func() managedartifactapply.Preparer { return managedartifact.NewCommitPreparer() }
var managedREADMEPusher = func() managedartifactapply.Pusher { return managedartifact.NewPusher() }
var managedREADMEJournalContext = func(configPath string) (string, managedartifactapply.JournalWriter, error) {
	executionID, writer, err := newJournalContext(configPath)
	return executionID, writer, err
}

func runApplyREADME(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(os.Stdout, "usage: repoctl apply-readme -f repora.yaml --plan-file FILE [--dry-run] [--json]")
		return 0
	}

	flags := flag.NewFlagSet("repoctl apply-readme", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("f", "repora.yaml", "path to SCHEMA-0001 YAML config")
	planFile := flags.String("plan-file", "", "exact repora.io/managed-artifact-plan v1 JSON")
	dryRun := flags.Bool("dry-run", false, "validate the exact plan and show its reviewed diff without mutation")
	jsonFlag := flags.Bool("json", false, "print stabilized managed README apply result JSON")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "repoctl: apply-readme does not accept positional arguments")
		return 1
	}
	if *planFile == "" {
		fmt.Fprintln(os.Stderr, "repoctl: apply-readme requires --plan-file FILE")
		return 1
	}
	if *dryRun && *jsonFlag {
		fmt.Fprintln(os.Stderr, "repoctl: apply-readme --json is supported for real apply only; use plan-readme --artifact for exact dry-run plan JSON")
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

	if *dryRun {
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

	if len(plan.Repositories) == 0 {
		fmt.Fprintln(os.Stdout, "No managed README changes.")
		return 0
	}
	executionID, writer, err := managedREADMEJournalContext(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: initialize managed README execution journal: %v\n", err)
		return 1
	}
	observer := managedREADMEPreflightObserver()
	result, executeErr := managedREADMEApplyExecute(
		spec,
		plan,
		observer,
		managedREADMECommitPreparer(),
		managedREADMEPusher(),
		managedartifactapply.Audit{ExecutionID: executionID, Writer: writer},
	)
	if *jsonFlag {
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write managed README apply JSON: %v\n", err)
			return 1
		}
	} else {
		printManagedREADMEApplyResult(result)
	}
	if executeErr != nil {
		fmt.Fprintf(os.Stderr, "repoctl: managed README apply: %v\n", executeErr)
		if errors.Is(executeErr, managedartifact.ErrStale) {
			return 2
		}
		return 1
	}
	return 0
}

func printManagedREADMEApplyResult(result managedartifactapply.Result) {
	for _, repo := range result.Repositories {
		fmt.Fprintf(os.Stdout, "%s (%s) %s %s", repo.ID, repo.UID, repo.Outcome, repo.Branch)
		if repo.CommitOID != "" {
			fmt.Fprintf(os.Stdout, " commit=%s", repo.CommitOID)
		}
		fmt.Fprintln(os.Stdout)
	}
	if result.Journal.Intent != "" {
		fmt.Fprintf(os.Stdout, "journal intent: %s\n", result.Journal.Intent)
	}
	if result.Journal.Result != "" {
		fmt.Fprintf(os.Stdout, "journal result: %s\n", result.Journal.Result)
	}
}
