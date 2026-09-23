package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"repoctl/internal/posture"
)

var collectLocalStoragePosture = posture.CollectLocalGitStorage

func runPostureStorage(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		printPostureStorageUsage(os.Stdout)
		return 0
	}
	flags := flag.NewFlagSet("repoctl posture storage", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	repository := flags.String("repository", "", "operator-asserted OWNER/REPO identity")
	path := flags.String("path", "", "existing local checkout or bare Git repository")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	if flags.NArg() != 0 || strings.TrimSpace(*repository) == "" || strings.TrimSpace(*path) == "" {
		printPostureStorageUsage(os.Stderr)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	inventory, err := collectLocalStoragePosture(ctx, *path, *repository)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: posture storage: %v\n", err)
		return 1
	}
	data, err := inventory.Marshal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: posture storage: %v\n", err)
		return 1
	}
	if _, err := os.Stdout.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: write storage posture inventory: %v\n", err)
		return 1
	}
	return 0
}

func printPostureStorageUsage(w *os.File) {
	fmt.Fprintln(w, "usage: repoctl posture storage --repository OWNER/REPO --path LOCAL_GIT_REPOSITORY")
}
