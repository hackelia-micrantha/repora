package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"repoctl/internal/bootstrap"
	"repoctl/internal/config"
	"repoctl/internal/posture"
)

var bootstrapDiscoverReader = func() posture.GitHubReader {
	return posture.NewHTTPGitHubReader(githubPostureToken())
}

func runBootstrap(args []string) int {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "-h" || args[0] == "--help")) {
		printBootstrapUsage(os.Stdout)
		return 0
	}
	switch args[0] {
	case "init":
		return runBootstrapInit(args[1:])
	case "inspect":
		return runBootstrapInspect(args[1:])
	case "plan":
		return runBootstrapPlan(args[1:])
	case "apply":
		return runBootstrapApply(args[1:])
	case "discover":
		return runBootstrapDiscover(args[1:])
	default:
		printBootstrapUsage(os.Stderr)
		return 1
	}
}

func runBootstrapInit(args []string) int {
	flags := flag.NewFlagSet("repoctl bootstrap init", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	root := flags.String("root", ".", "repository workspace root")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "repoctl: bootstrap init does not accept positional arguments")
		return 1
	}
	path := filepath.Join(*root, filepath.FromSlash(bootstrap.DefaultProposalPath))
	if err := bootstrap.Init(path); err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: bootstrap init: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "created %s\n", path)
	fmt.Fprintf(os.Stdout, "contract %s@%s\n", bootstrap.ContractRepository, bootstrap.ContractRevision)
	return 0
}

func runBootstrapInspect(args []string) int {
	flags := flag.NewFlagSet("repoctl bootstrap inspect", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	root := flags.String("root", ".", "repository workspace root")
	jsonFlag := flags.Bool("json", false, "print versioned inspection JSON")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "repoctl: bootstrap inspect does not accept positional arguments")
		return 1
	}
	inspection, err := bootstrap.Inspect(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: bootstrap inspect: %v\n", err)
		return 1
	}
	if *jsonFlag {
		return writeJSON(inspection)
	}
	fmt.Fprintf(os.Stdout, "root: %s\n", inspection.Root)
	fmt.Fprintf(os.Stdout, "snapshot: %s\n", inspection.SnapshotSHA256)
	for _, observation := range inspection.Observations {
		fmt.Fprintf(os.Stdout, "%s: %t", observation.Key, observation.Value)
		if observation.SHA256 != "" {
			fmt.Fprintf(os.Stdout, " sha256=%s", observation.SHA256)
		}
		fmt.Fprintln(os.Stdout)
	}
	return 0
}

func runBootstrapPlan(args []string) int {
	flags := flag.NewFlagSet("repoctl bootstrap plan", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	root := flags.String("root", ".", "repository workspace root")
	manifestPath := flags.String("manifest", "", "bootstrap proposal manifest (default ROOT/.repora/bootstrap.proposed.json)")
	artifact := flags.Bool("artifact", false, "print exact repora.bootstrap-plan v1 JSON")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "repoctl: bootstrap plan does not accept positional arguments")
		return 1
	}
	manifest := *manifestPath
	if manifest == "" {
		manifest = filepath.Join(*root, filepath.FromSlash(bootstrap.DefaultProposalPath))
	}
	plan, err := bootstrap.BuildPlan(*root, manifest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: bootstrap plan: %v\n", err)
		return 1
	}
	if *artifact {
		data, err := plan.Marshal()
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: bootstrap plan: %v\n", err)
			return 1
		}
		_, _ = os.Stdout.Write(data)
		return 0
	}
	printBootstrapPlan(plan)
	return 0
}

func runBootstrapApply(args []string) int {
	flags := flag.NewFlagSet("repoctl bootstrap apply", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	planFile := flags.String("plan-file", "", "exact repora.bootstrap-plan v1 JSON")
	jsonFlag := flags.Bool("json", false, "print versioned apply result JSON")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	if flags.NArg() != 0 || *planFile == "" {
		fmt.Fprintln(os.Stderr, "usage: repoctl bootstrap apply --plan-file FILE [--json]")
		return 1
	}
	data, err := os.ReadFile(*planFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: bootstrap apply: read plan: %v\n", err)
		return 1
	}
	plan, err := bootstrap.ParsePlan(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: bootstrap apply: %v\n", err)
		return 1
	}
	result, err := bootstrap.Apply(plan)
	if *jsonFlag || result.Kind != "" {
		if encodeErr := json.NewEncoder(os.Stdout).Encode(result); encodeErr != nil {
			fmt.Fprintf(os.Stderr, "repoctl: bootstrap apply: write result: %v\n", encodeErr)
			return 1
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: bootstrap apply: %v\n", err)
		if errors.Is(err, bootstrap.ErrStale) {
			return 2
		}
		return 1
	}
	return 0
}

func runBootstrapDiscover(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(os.Stdout, "usage: repoctl bootstrap discover -f repora.yaml [--previous FILE] OWNER/REPO")
		return 0
	}
	flags := flag.NewFlagSet("repoctl bootstrap discover", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("f", "repora.yaml", "path to SCHEMA-0001 YAML config")
	previousPath := flags.String("previous", "", "previous repora.bootstrap-discovery v1 JSON for rescan classification")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	if flags.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: repoctl bootstrap discover -f repora.yaml [--previous FILE] OWNER/REPO")
		return 1
	}
	spec, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: bootstrap discover: %v\n", err)
		return 1
	}
	var previous *bootstrap.Discovery
	if *previousPath != "" {
		data, err := os.ReadFile(*previousPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: bootstrap discover: read previous: %v\n", err)
			return 1
		}
		parsed, err := bootstrap.ParseDiscovery(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: bootstrap discover: %v\n", err)
			return 1
		}
		previous = &parsed
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	discovery, err := bootstrap.Discover(ctx, bootstrapDiscoverReader(), spec, flags.Arg(0), previous)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: bootstrap discover: %v\n", err)
		return 1
	}
	data, err := discovery.Marshal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: bootstrap discover: %v\n", err)
		return 1
	}
	_, _ = os.Stdout.Write(data)
	return 0
}

func printBootstrapPlan(plan bootstrap.Plan) {
	fmt.Fprintf(os.Stdout, "contract: %s@%s\n", plan.Contract.Repository, plan.Contract.Revision)
	fmt.Fprintf(os.Stdout, "snapshot: %s\n", plan.SnapshotSHA256)
	fmt.Fprintf(os.Stdout, "ci applicability: %s\n", plan.CIApplicability)
	if len(plan.Actions) == 0 {
		fmt.Fprintln(os.Stdout, "ready actions: none")
	} else {
		fmt.Fprintln(os.Stdout, "ready actions:")
		for _, action := range plan.Actions {
			fmt.Fprintf(os.Stdout, "  %s %s\n", action.Type, action.Target)
		}
	}
	if len(plan.BlockedActions) > 0 {
		fmt.Fprintln(os.Stdout, "blocked actions:")
		for _, action := range plan.BlockedActions {
			fmt.Fprintf(os.Stdout, "  %s", action.Type)
			if action.Target != "" {
				fmt.Fprintf(os.Stdout, " %s", action.Target)
			}
			fmt.Fprintln(os.Stdout)
			for _, reason := range action.Reasons {
				fmt.Fprintf(os.Stdout, "    - %s\n", reason)
			}
		}
	}
	if len(plan.UnresolvedKeys) > 0 {
		fmt.Fprintln(os.Stdout, "unresolved decisions:")
		for _, key := range plan.UnresolvedKeys {
			fmt.Fprintf(os.Stdout, "  %s\n", key)
		}
	}
}

func writeJSON(value interface{}) int {
	if err := json.NewEncoder(os.Stdout).Encode(value); err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: write json: %v\n", err)
		return 1
	}
	return 0
}

func printBootstrapUsage(w *os.File) {
	fmt.Fprintln(w, "usage: repoctl bootstrap init [--root DIR]")
	fmt.Fprintln(w, "       repoctl bootstrap inspect [--root DIR] [--json]")
	fmt.Fprintln(w, "       repoctl bootstrap plan [--root DIR] [--manifest FILE] [--artifact]")
	fmt.Fprintln(w, "       repoctl bootstrap apply --plan-file FILE [--json]")
	fmt.Fprintln(w, "       repoctl bootstrap discover -f repora.yaml [--previous FILE] OWNER/REPO")
}
