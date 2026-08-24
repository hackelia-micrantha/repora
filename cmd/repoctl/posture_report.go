package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"repoctl/internal/posturepolicy"
)

func runPostureReport(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		printPostureReportUsage(os.Stdout)
		return 0
	}
	flags := flag.NewFlagSet("repoctl posture report", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	profilePath := flags.String("profile", "", "path to repora.posture-policy-profile v1 JSON")
	factsPath := flags.String("facts", "", "path to repora.posture-policy-inputs v1 JSON")
	format := flags.String("format", "markdown", "output format: markdown or json")
	asOfValue := flags.String("as-of", "", "policy evaluation date in YYYY-MM-DD format")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	if flags.NArg() != 0 || *profilePath == "" || *factsPath == "" || *asOfValue == "" {
		printPostureReportUsage(os.Stderr)
		return 1
	}
	asOf, err := time.Parse("2006-01-02", *asOfValue)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: posture report: --as-of must use YYYY-MM-DD: %v\n", err)
		return 1
	}
	profileData, err := os.ReadFile(*profilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: posture report: read profile: %v\n", err)
		return 1
	}
	profile, err := posturepolicy.ParseProfile(profileData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: posture report: %v\n", err)
		return 1
	}
	factsData, err := os.ReadFile(*factsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: posture report: read facts: %v\n", err)
		return 1
	}
	inputs, err := posturepolicy.ParseInputs(factsData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: posture report: %v\n", err)
		return 1
	}
	report, err := posturepolicy.Evaluate(profile, inputs, asOf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: posture report: %v\n", err)
		return 1
	}
	switch *format {
	case "markdown":
		if _, err := fmt.Fprint(os.Stdout, posturepolicy.RenderMarkdown(report)); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write posture report: %v\n", err)
			return 1
		}
	case "json":
		data, err := report.Marshal()
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: posture report: %v\n", err)
			return 1
		}
		if _, err := os.Stdout.Write(data); err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: write posture report: %v\n", err)
			return 1
		}
	default:
		fmt.Fprintln(os.Stderr, "repoctl: posture report: --format must be markdown or json")
		return 1
	}
	return 0
}

func printPostureReportUsage(w *os.File) {
	fmt.Fprintln(w, "usage: repoctl posture report --profile POLICY.json --facts FACTS.json --as-of YYYY-MM-DD [--format markdown|json]")
}
