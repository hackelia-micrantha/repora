package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"repoctl/internal/config"
	"repoctl/internal/posture"
)

var collectGitHubPosture = func(ctx context.Context, fullName, token string) (posture.Inventory, error) {
	return posture.CollectGitHub(ctx, posture.NewHTTPGitHubReader(token), fullName)
}

var collectGitHubDocumentationPosture = func(ctx context.Context, fullName, token string) (posture.DocumentationInventory, error) {
	return posture.CollectGitHubDocumentation(ctx, posture.NewHTTPGitHubReader(token), fullName)
}

var collectGitHubHooksPosture = func(ctx context.Context, fullName, token string) (posture.HooksInventory, error) {
	return posture.CollectGitHubHooks(ctx, posture.NewHTTPGitHubReader(token), fullName)
}

var collectGitHubCommitPosture = func(ctx context.Context, fullName, token string) (posture.CommitInventory, error) {
	reader := posture.NewHTTPGitHubReader(token)
	return posture.CollectGitHubCommits(ctx, reader, reader, fullName)
}

var collectMirrorPosture = func(ctx context.Context, spec config.Spec, token string) (posture.MirrorInventory, error) {
	return posture.CollectMirrorPosture(ctx, spec, posture.GitMirrorLocalObserver{}, posture.DefaultMirrorProviderReader{
		GitHub: posture.NewHTTPGitHubReader(token),
	})
}

func runPosture(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		printPostureUsage(os.Stdout)
		return 0
	}
	if len(args) > 0 && args[0] == "mirrors" {
		return runMirrorPosture(args[1:])
	}
	if len(args) > 0 && args[0] == "report" {
		return runPostureReport(args[1:])
	}
	if len(args) == 2 && (args[1] == "-h" || args[1] == "--help") {
		switch args[0] {
		case "inventory", "docs", "hooks", "commits":
			fmt.Fprintf(os.Stdout, "usage: repoctl posture %s OWNER/REPO\n", args[0])
			return 0
		}
	}
	if len(args) != 2 {
		printPostureUsage(os.Stderr)
		return 1
	}

	token := githubPostureToken()
	timeout := 45 * time.Second
	if args[0] == "commits" {
		timeout = 90 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
	case "hooks":
		inventory, err := collectGitHubHooksPosture(ctx, args[1], token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: posture hooks: %v\n", err)
			return 1
		}
		data, err := inventory.Marshal()
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: posture hooks: %v\n", err)
			return 1
		}
		if _, err := os.Stdout.Write(data); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write hooks posture inventory: %v\n", err)
			return 1
		}
		return 0
	case "commits":
		inventory, err := collectGitHubCommitPosture(ctx, args[1], token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: posture commits: %v\n", err)
			return 1
		}
		data, err := inventory.Marshal()
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: posture commits: %v\n", err)
			return 1
		}
		if _, err := os.Stdout.Write(data); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write commit posture inventory: %v\n", err)
			return 1
		}
		return 0
	default:
		printPostureUsage(os.Stderr)
		return 1
	}
}

func runMirrorPosture(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(os.Stdout, "usage: repoctl posture mirrors -f repora.yaml")
		return 0
	}
	flags := flag.NewFlagSet("repoctl posture mirrors", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("f", "repora.yaml", "path to SCHEMA-0001 YAML config")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: repoctl posture mirrors -f repora.yaml")
		return 1
	}
	spec, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: posture mirrors: %v\n", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	inventory, err := collectMirrorPosture(ctx, spec, githubPostureToken())
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: posture mirrors: %v\n", err)
		return 1
	}
	data, err := inventory.Marshal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: posture mirrors: %v\n", err)
		return 1
	}
	if _, err := os.Stdout.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: write mirror posture inventory: %v\n", err)
		return 1
	}
	return 0
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
	fmt.Fprintln(w, "       repoctl posture hooks OWNER/REPO")
	fmt.Fprintln(w, "       repoctl posture commits OWNER/REPO")
	fmt.Fprintln(w, "       repoctl posture mirrors -f repora.yaml")
}
