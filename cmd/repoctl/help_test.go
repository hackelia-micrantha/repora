package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestIsHelpRequest(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"help"}, {"-h"}, {"--help"}} {
		if !isHelpRequest(args) {
			t.Fatalf("isHelpRequest(%q) = false, want true", args)
		}
	}

	for _, args := range [][]string{nil, {}, {"status"}, {"status", "--help"}, {"--help", "extra"}} {
		if isHelpRequest(args) {
			t.Fatalf("isHelpRequest(%q) = true, want false", args)
		}
	}
}

func TestRunHelp(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"-h"}, {"--help"}} {
		args := args
		t.Run(args[0], func(t *testing.T) {
			var output bytes.Buffer
			code := withStdout(t, &output, func() int {
				return run(args)
			})

			if code != 0 {
				t.Fatalf("run(%q) = %d, want 0", args, code)
			}
			if !strings.Contains(output.String(), "Usage:") {
				t.Fatalf("help output missing Usage:\n%s", output.String())
			}
		})
	}
}

func TestPrintHelp(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	printHelp(&output)

	text := output.String()
	for _, expected := range []string{
		"Usage:",
		"repoctl --help",
		"status",
		"plan",
		"apply",
		"sync     alias for apply",
		"validate-report",
		"list-findings",
		"generate-scorecard",
		"assess",
		"-h, --help",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("help output missing %q:\n%s", expected, text)
		}
	}
}
