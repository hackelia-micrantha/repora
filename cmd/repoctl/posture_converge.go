package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"repoctl/internal/posturepolicy"
)

type singlePathFlag struct {
	name  string
	value string
	set   bool
}

func (f *singlePathFlag) String() string {
	return f.value
}

func (f *singlePathFlag) Set(value string) error {
	if f.set {
		return fmt.Errorf("--%s may only be specified once", f.name)
	}
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("--%s requires a non-empty path", f.name)
	}
	f.value = value
	f.set = true
	return nil
}

func runPostureConverge(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		printPostureConvergeUsage(os.Stdout)
		return 0
	}

	flags := flag.NewFlagSet("repoctl posture converge", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	inventoryPath := singlePathFlag{name: "inventory"}
	documentationPath := singlePathFlag{name: "docs"}
	hooksPath := singlePathFlag{name: "hooks"}
	commitsPath := singlePathFlag{name: "commits"}
	mirrorsPath := singlePathFlag{name: "mirrors"}
	flags.Var(&inventoryPath, "inventory", "path to repora.posture-inventory v1 JSON")
	flags.Var(&documentationPath, "docs", "path to repora.posture-documentation v1 JSON")
	flags.Var(&hooksPath, "hooks", "path to repora.posture-hooks v1 JSON")
	flags.Var(&commitsPath, "commits", "path to repora.posture-commits v1 JSON")
	flags.Var(&mirrorsPath, "mirrors", "path to repora.posture-mirrors v1 JSON")
	mirrorRepoUID := flags.String("repo-uid", "", "repository uid to select from --mirrors")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	if flags.NArg() != 0 {
		printPostureConvergeUsage(os.Stderr)
		return 1
	}
	if !inventoryPath.set && !documentationPath.set && !hooksPath.set && !commitsPath.set && !mirrorsPath.set {
		printPostureConvergeUsage(os.Stderr)
		return 1
	}
	if mirrorsPath.set != (strings.TrimSpace(*mirrorRepoUID) != "") {
		fmt.Fprintln(os.Stderr, "repoctl: posture converge: --mirrors and --repo-uid must be supplied together")
		return 1
	}

	artifacts := posturepolicy.ArtifactSet{MirrorRepoUID: *mirrorRepoUID}
	var err error
	if inventoryPath.set {
		artifacts.Inventory, err = readPostureArtifact("inventory", inventoryPath.value)
		if err != nil {
			return 1
		}
	}
	if documentationPath.set {
		artifacts.Documentation, err = readPostureArtifact("docs", documentationPath.value)
		if err != nil {
			return 1
		}
	}
	if hooksPath.set {
		artifacts.Hooks, err = readPostureArtifact("hooks", hooksPath.value)
		if err != nil {
			return 1
		}
	}
	if commitsPath.set {
		artifacts.Commits, err = readPostureArtifact("commits", commitsPath.value)
		if err != nil {
			return 1
		}
	}
	if mirrorsPath.set {
		artifacts.Mirrors, err = readPostureArtifact("mirrors", mirrorsPath.value)
		if err != nil {
			return 1
		}
	}

	inputs, err := posturepolicy.ConvergeArtifacts(artifacts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: posture converge: %v\n", err)
		return 1
	}
	data, err := inputs.Marshal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: posture converge: %v\n", err)
		return 1
	}
	if _, err := os.Stdout.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: write posture policy inputs: %v\n", err)
		return 1
	}
	return 0
}

func readPostureArtifact(name, path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: posture converge: read %s artifact: %v\n", name, err)
		return nil, err
	}
	return data, nil
}

func printPostureConvergeUsage(w *os.File) {
	fmt.Fprintln(w, "usage: repoctl posture converge [--inventory FILE] [--docs FILE] [--hooks FILE] [--commits FILE] [--mirrors FILE --repo-uid UID]")
}
