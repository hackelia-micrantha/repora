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

var collectGitHubDocumentationPosture = func(ctx context.Context, fullName, token string) (posture.DocumentationInventory, error) {
	return posture.CollectGitHubDocumentation(ctx, posture.NewHTTPGitHubReader(token), fullName)
}

func runPosture(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		printPostureUsage(os.Stdout)
		return 0
	}
	if len(args) == 2 && (args[1] == "-h" || args[1] == "--help") {
		switch args[0] {
		case "inventory", "docs":
			fmt.Fprintf(os.Stdout, "usage: repoctl posture %s OWNER/REPO\n", args[0])
			return 0
		}
	}
	if len(args) != 2 {
		printPostureUsage(os.Stderr)
		return 1
	}

	token := githubPostureToken()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	switch args[0] {
	case "inventory":
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
	case "docs":
		inventory, err := collectGitHubDocumentationPosture(ctx, args[1], token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: posture docs: %v\n", err)
			return 1
		}
		data, err := inventory.Marshal()
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: posture docs: %v\n", err)
			return 1
		}
		if _, err := os.Stdout.Write(data); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write documentation posture inventory: %v\n", err)
			return 1
		}
		return 0
	default:
		printPostureUsage(os.Stderr)
		return 1
	}
}

func githubPostureToken() string {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}
	return os.Getenv("GH_TOKEN")
}

func printPostureUsage(w *os.File) {
	fmt.Fprintln(w, "usage: repoctl posture inventory OWNER/REPO")
	fmt.Fprintln(w, "       repoctl posture docs OWNER/REPO")
}
