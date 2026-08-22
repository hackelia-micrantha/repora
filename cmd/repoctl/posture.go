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

func runMirrorPosture(args []string) int {
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
	fmt.Fprintln(w, "       repoctl posture mirrors -f repora.yaml")
}
