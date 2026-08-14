package main

import (
	"fmt"
	"io"
)

const helpText = `repoctl manages configured Git repository state and read-only repository evidence.

Usage:
  repoctl <command> [options]
  repoctl posture inventory OWNER/REPO
  repoctl plan-readme -f repora.yaml [--artifact]
  repoctl apply-readme -f repora.yaml --plan-file FILE [--dry-run] [--json]
  repoctl validate-report FILE
  repoctl list-findings FILE
  repoctl generate-scorecard FILE
  repoctl assess FILE
  repoctl --help
  repoctl --version

Commands:
  status   inspect canonical and mirror default-branch state
  plan     show planned mirror updates or export an executable artifact
  apply    apply current observations or an exact plan artifact
  sync     alias for apply
  posture  collect read-only repository/CI posture facts; inventory is the current subcommand
  plan-readme  review managed README changes or export the exact managed-artifact plan
  apply-readme  dry-run or journaled exact-plan managed README apply
  validate-report  validate a repository assessment report without mutation
  list-findings    list validated assessment findings without mutation
  generate-scorecard  render validated scorecard dimensions without recalculation
  assess   create a new canonical assessment skeleton without overwriting files
  version  show embedded version and commit metadata
  help     show this help

Options for mirror commands:
  -f string
        path to SCHEMA-0001 YAML config (default "repora.yaml")
  --json
        print stabilized command JSON
  --artifact
        with plan, print the exact executable plan artifact as JSON
  --plan-file string
        with apply or sync, execute the exact artifact from this file
  --parallel int
        maximum concurrent repository operations (default 5)
  --continue-on-error
        continue processing repositories after an error
  --dry-run
        show what apply or sync would change without mutation
  --force
        allow destructive overwrites for ahead or diverged mirrors
  --debug
        print debug logs to stderr

Options for posture inventory:
  OWNER/REPO
        GitHub repository to inspect. Public repositories need no token; private/provider-protected evidence may use GITHUB_TOKEN or GH_TOKEN from the environment.

Posture inventory is GET-only. Provider fields unavailable under current access are emitted as unavailable facts rather than false negatives. It does not evaluate policy, create findings, run scanners, or mutate repository/provider state.

Options for plan-readme:
  -f string
        path to SCHEMA-0001 YAML config (default "repora.yaml")
  --artifact
        print the exact repora.io/managed-artifact-plan v1 JSON instead of human review output

Options for apply-readme:
  -f string
        path to SCHEMA-0001 YAML config (default "repora.yaml")
  --plan-file string
        exact repora.io/managed-artifact-plan v1 JSON to preflight or apply
  --dry-run
        revalidate current canonical state and show the reviewed diff without mutation
  --json
        on real apply, print repora.io/managed-artifact-apply-result v1 JSON

A real apply persists managed-artifact INTENT before candidate creation, creates verified local candidate commits, performs fresh stale preflight, pushes with an exact reviewed-base lease, and persists RESULT evidence. It does not use --force.

Global options:
  --version
        show embedded version and commit metadata
  -h, --help
        show this help
`

func isHelpRequest(args []string) bool {
	if len(args) != 1 {
		return false
	}

	switch args[0] {
	case "help", "-h", "--help":
		return true
	default:
		return false
	}
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, helpText)
}
