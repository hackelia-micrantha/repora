package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"repoctl/internal/config"
	"repoctl/internal/managedartifact"
)

var managedREADMEPlanBuild = managedartifact.BuildPlan
var managedREADMEObserver = func() managedartifact.READMEObserver { return managedartifact.NewGitREADMEObserver() }

func runPlanREADME(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(os.Stdout, "usage: repoctl plan-readme -f repora.yaml [--artifact]")
		return 0
	}

	flags := flag.NewFlagSet("repoctl plan-readme", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("f", "repora.yaml", "path to SCHEMA-0001 YAML config")
	artifact := flags.Bool("artifact", false, "print the exact managed-artifact plan as JSON")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "repoctl: plan-readme does not accept positional arguments")
		return 1
	}

	spec, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: %v\n", err)
		return 1
	}
	plan, err := managedREADMEPlanBuild(*configPath, spec, managedREADMEObserver())
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: build managed README plan: %v\n", err)
		return 1
	}
	if err := plan.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: validate managed README plan: %v\n", err)
		return 1
	}
	if *artifact {
		data, err := plan.Marshal()
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: serialize managed README plan: %v\n", err)
			return 1
		}
		data = append(data, '\n')
		if _, err := os.Stdout.Write(data); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write managed README plan: %v\n", err)
			return 1
		}
		return 0
	}
	printManagedREADMEPlan(os.Stdout, plan)
	return 0
}

func printManagedREADMEPlan(w io.Writer, plan managedartifact.Plan) {
	if len(plan.Repositories) == 0 {
		fmt.Fprintln(w, "No managed README changes.")
		return
	}
	for i, repo := range plan.Repositories {
		if i > 0 {
			fmt.Fprintln(w)
		}
		action := repo.Actions[0]
		fmt.Fprintf(w, "%s (%s) %s/%s#%s\n", repo.ID, repo.UID, repo.Target.Provider, repo.Target.Path, repo.Target.Branch)
		fmt.Fprintf(w, "base: %s\n", repo.BaseOID)
		if action.Observed.Present != nil && *action.Observed.Present {
			fmt.Fprintf(w, "README.md: %s %s -> %s %s\n", action.Observed.Mode, action.Observed.SHA256, action.Desired.Mode, action.Desired.SHA256)
		} else {
			fmt.Fprintf(w, "README.md: absent -> %s %s\n", action.Desired.Mode, action.Desired.SHA256)
		}
		fmt.Fprint(w, action.Diff)
	}
}
