package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"repoctl/internal/posture"
)

var collectGitHubPosture = func(ctx context.Context, fullName, token string) (posture.Inventory, error) {
	return posture.CollectGitHub(ctx, posture.NewHTTPGitHubReader(token), fullName)
}

func runPosture(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(os.Stdout, "usage: repoctl posture inventory OWNER/REPO")
		return 0
	}
	if len(args) == 2 && args[0] == "inventory" && (args[1] == "-h" || args[1] == "--help") {
		fmt.Fprintln(os.Stdout, "usage: repoctl posture inventory OWNER/REPO")
		return 0
	}
	if len(args) != 2 || args[0] != "inventory" {
		fmt.Fprintln(os.Stderr, "usage: repoctl posture inventory OWNER/REPO")
		return 1
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	inventory, err := collectGitHubPosture(ctx, args[1], token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: posture inventory: %v\n", err)
		return 1
	}
	data, err := inventory.Marshal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: posture inventory: %v\n", err)
		return 1
	}
	if _, err := os.Stdout.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: write posture inventory: %v\n", err)
		return 1
	}
	return 0
}
