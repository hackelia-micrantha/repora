package main

import (
	"fmt"
	"io"
)

const helpText = `repoctl manages configured Git repository mirrors.

Usage:
  repoctl <command> [options]
  repoctl --help

Commands:
  status   inspect canonical and mirror default-branch state
  plan     show planned mirror updates or export an executable artifact
  apply    apply current observations or an exact plan artifact
  sync     alias for apply
  help     show this help

Options:
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
