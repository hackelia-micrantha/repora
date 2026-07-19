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
  plan     show planned mirror updates
  apply    apply planned default-branch updates
  sync     alias for apply
  help     show this help

Options:
  -f string
        path to SCHEMA-0001 YAML config (default "repora.yaml")
  --json
        print JSON
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
