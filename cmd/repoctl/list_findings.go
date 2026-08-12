package main

import (
	"fmt"
	"os"
	"strconv"
)

func runListFindings(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(os.Stdout, "usage: repoctl list-findings FILE")
		return 0
	}
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: repoctl list-findings FILE")
		return 1
	}

	report, err := loadAssessmentReport(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: %v\n", err)
		return 1
	}

	for _, finding := range report.Findings {
		fmt.Fprintf(
			os.Stdout,
			"%s\t%s\t%s\t%s\t%s\n",
			finding.ID,
			finding.Severity,
			finding.Status,
			finding.Type,
			strconv.Quote(finding.Title),
		)
	}
	return 0
}
