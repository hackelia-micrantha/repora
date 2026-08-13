package main

import (
	"flag"
	"fmt"
	"os"

	"repoctl/internal/apply"
	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

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

var planBuild = func(repo config.Repo, result status.Result, _ bool) (planartifact.Artifact, error) {
	return apply.BuildArtifact(repo, result, gitwrap.Client{})
}

var applyExecute = func(repo config.Repo, result status.Result, force, dryRun bool) (apply.Result, error) {
	return apply.Execute(repo, result, gitwrap.Client{}, force, dryRun)
}

var artifactApplyExecute = func(repo config.Repo, result status.Result, artifact planartifact.Artifact, force, dryRun bool) (apply.Result, error) {
	return apply.ExecuteArtifact(repo, result, artifact, gitwrap.Client{}, force, dryRun)
}

var auditedApplyExecute = func(repo config.Repo, result status.Result, force, dryRun bool, audit apply.Audit) (apply.Result, error) {
	return apply.ExecuteAudited(repo, result, gitwrap.Client{}, force, dryRun, audit)
}

var auditedArtifactApplyExecute = func(repo config.Repo, result status.Result, artifact planartifact.Artifact, force, dryRun bool, audit apply.Audit) (apply.Result, error) {
	return apply.ExecuteArtifactAudited(repo, result, artifact, force, dryRun, audit)
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
	if isVersionRequest(args) {
		printVersion(os.Stdout)
		return 0
	}
	if isHelpRequest(args) {
		printHelp(os.Stdout)
		return 0
	}

	if len(args) == 0 {
		printUsageError()
		return 1
	}
	if args[0] == "validate-report" {
		return runValidateReport(args[1:])
	}
	if args[0] == "list-findings" {
		return runListFindings(args[1:])
	}
	if args[0] == "generate-scorecard" {
		return runGenerateScorecard(args[1:])
	}
	if args[0] == "assess" {
		return runAssess(args[1:])
	}
	if args[0] == "plan-readme" {
		return runPlanREADME(args[1:])
	}
	if args[0] == "apply-readme" {
		return runApplyREADME(args[1:])
	}

	command := args[0]
	if command != "status" && command != "plan" && command != "apply" && command != "sync" {
		printUsageError()
		return 1
	}

	flags := flag.NewFlagSet("repoctl "+command, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("f", "repora.yaml", "path to SCHEMA-0001 YAML config")
	jsonFlag := flags.Bool("json", false, "print stabilized command JSON")
	artifactFlag := flags.Bool("artifact", false, "print the exact executable plan artifact as JSON")
	planFile := flags.String("plan-file", "", "execute an exact plan artifact from this file")
	parallelFlag := flags.Int("parallel", 5, "max number of concurrent repository operations")
	continueOnError := flags.Bool("continue-on-error", false, "continue processing repos after an error")
	dryRun := flags.Bool("dry-run", false, "show what would change without mutating mirror state")
	force := flags.Bool("force", false, "allow destructive mirror overwrites for ahead or diverged mirrors")
	debug := flags.Bool("debug", false, "print debug logs to stderr")

	if err := flags.Parse(args[1:]); err != nil {
		return 1
	}
	if *artifactFlag && command != "plan" {
		fmt.Fprintln(os.Stderr, "repoctl: --artifact is only supported for plan")
		return 1
	}
	if *artifactFlag && *jsonFlag {
		fmt.Fprintln(os.Stderr, "repoctl: --artifact and --json are mutually exclusive")
		return 1
	}
	if *planFile != "" && command != "apply" && command != "sync" {
		fmt.Fprintln(os.Stderr, "repoctl: --plan-file is only supported for apply and sync")
		return 1
	}

	spec, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: %v\n", err)
		return 1
	}

	var inputArtifact *planartifact.Artifact
	if *planFile != "" {
		data, err := os.ReadFile(*planFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: read plan artifact: %v\n", err)
			return 1
		}
		artifact, err := planartifact.Parse(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: %v\n", err)
			return 1
		}
		selected, err := selectArtifactSpec(spec, artifact)
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: %v\n", err)
			return 1
		}
		spec = selected
		inputArtifact = &artifact
	}

	parallel := *parallelFlag
	if parallel < 1 {
		parallel = 1
	}

	if *debug {
		debugf("repoctl: debug command=%s repos=%d parallel=%d dry_run=%t\n", command, len(spec.Repos), parallel, *dryRun)
	}

	if command == "status" {
		return runMultiStatus(spec, parallel, *jsonFlag, *continueOnError, *debug)
	}
	if command == "plan" && hasMultiMirrorRepository(spec) {
		return runMultiMirrorPlan(spec, parallel, *jsonFlag, *artifactFlag, *force, *continueOnError, *debug)
	}
	if (command == "apply" || command == "sync") && hasMultiMirrorRepository(spec) {
		return runPathBoundApply(spec, parallel, *jsonFlag, *force, *dryRun, inputArtifact, *configPath, *debug)
	}
	if err := requireSingleMirrorReconciliation(spec); err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: %v\n", err)
		return 1
	}

	summary := checkRepos(spec, parallel, *debug)
	if summary.firstErr != nil && !*continueOnError {
		fmt.Fprintf(os.Stderr, "repoctl: %v\n", summary.firstErr)
		return 1
	}

	switch command {
	case "plan":
		return runPlan(spec, summary, *jsonFlag, *artifactFlag, *force)
	case "apply", "sync":
		return runApply(spec, summary, *jsonFlag, *force, *dryRun, parallel, inputArtifact, *configPath)
	default:
		panic("unreachable command validation")
	}
}

func printUsageError() {
	fmt.Fprintln(os.Stderr, "usage: repoctl <status|plan|apply|sync> -f repora.yaml [--json|--artifact] [--plan-file FILE] [--parallel N] [--continue-on-error] [--dry-run] [--force] [--debug]")
	fmt.Fprintln(os.Stderr, "       repoctl plan-readme -f repora.yaml [--artifact]")
	fmt.Fprintln(os.Stderr, "       repoctl apply-readme -f repora.yaml --plan-file FILE --dry-run")
	fmt.Fprintln(os.Stderr, "       repoctl validate-report FILE")
	fmt.Fprintln(os.Stderr, "       repoctl list-findings FILE")
	fmt.Fprintln(os.Stderr, "       repoctl generate-scorecard FILE")
	fmt.Fprintln(os.Stderr, "       repoctl assess FILE")
}
