package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"repoctl/internal/assessment"
)

func runAssess(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(os.Stdout, "usage: repoctl assess FILE")
		return 0
	}
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: repoctl assess FILE")
		return 1
	}

	report := assessment.NewSkeleton()
	if err := report.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: generated assessment template is invalid: %v\n", err)
		return 1
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: encode assessment template: %v\n", err)
		return 1
	}
	data = append(data, '\n')

	if err := writeNewAssessmentFile(args[0], data); err != nil {
		fmt.Fprintf(os.Stderr, "repoctl: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "created assessment template %s\n", args[0])
	return 0
}

func writeNewAssessmentFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("assessment report already exists: %s", path)
		}
		return fmt.Errorf("create assessment report: %w", err)
	}

	written, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return fmt.Errorf("write assessment report: %w; partially created file retained at %s", writeErr, path)
	}
	if written != len(data) {
		return fmt.Errorf("write assessment report: short write %d of %d bytes; partially created file retained at %s", written, len(data), path)
	}
	if closeErr != nil {
		return fmt.Errorf("close assessment report: %w; created file retained at %s", closeErr, path)
	}
	return nil
}
