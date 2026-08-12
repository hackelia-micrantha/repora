package main

import (
	"fmt"
	"io"
)

const helpText = `repoctl manages configured Git repository mirrors and validates Repora report artifacts.

Usage:
  repoctl <command> [options]
  repoctl validate-report FILE
  repoctl list-findings FILE
  repoctl --help
  repoctl --version

Commands:
  status   inspect canonical and mirror default-branch state
  plan     show planned mirror updates or export an executable artifact
  apply    apply current observations or an exact plan artifact
  sync     alias for apply
  validate-report  validate a repository assessment report without mutation
  list-findings    list validated assessment findings without mutation
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
