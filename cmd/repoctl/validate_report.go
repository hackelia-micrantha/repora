package main

import (
	"fmt"
	"os"

	"repoctl/internal/assessment"
)

func runValidateReport(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(os.Stdout, "usage: repoctl validate-report FILE")
		return 0
	}
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: repoctl validate-report FILE")
		return 1
	}

	report, err := loadAssessmentReport(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "valid assessment %s (%s)\n", report.ID, report.Snapshot.Revision.Commit)
	return 0
}

func loadAssessmentReport(path string) (assessment.Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return assessment.Report{}, fmt.Errorf("read assessment report: %w", err)
	}
	report, err := assessment.Parse(data)
	if err != nil {
		return assessment.Report{}, err
	}
	return report, nil
}
