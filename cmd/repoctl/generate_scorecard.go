package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func runGenerateScorecard(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(os.Stdout, "usage: repoctl generate-scorecard FILE")
		return 0
	}
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: repoctl generate-scorecard FILE")
		return 1
	}

	report, err := loadAssessmentReport(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: %v\n", err)
		return 1
	}

	for _, dimension := range report.Scorecard.Dimensions {
		evidenceIDs, err := json.Marshal(dimension.EvidenceIDs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: encode scorecard %s evidence ids: %v\n", dimension.Name, err)
			return 1
		}
		rationale, err := json.Marshal(dimension.Rationale)
		if err != nil {
			fmt.Fprintf(os.Stderr, "repoctl: encode scorecard %s rationale: %v\n", dimension.Name, err)
			return 1
		}
		fmt.Fprintf(
			os.Stdout,
			"%s\t%d\t%s\t%s\n",
			dimension.Name,
			*dimension.Score,
			evidenceIDs,
			rationale,
		)
	}
	return 0
}
