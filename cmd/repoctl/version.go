package main

import (
	"fmt"
	"io"
)

var (
	version = "dev"
	commit  = "unknown"
)

func isVersionRequest(args []string) bool {
	return len(args) == 1 && (args[0] == "version" || args[0] == "--version")
}

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "repoctl %s (%s)\n", version, commit)
}
