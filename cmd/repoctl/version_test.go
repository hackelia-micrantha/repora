package main

import (
	"bytes"
	"testing"
)

func TestVersionRequest(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"--version"}} {
		if !isVersionRequest(args) {
			t.Fatalf("isVersionRequest(%q) = false", args)
		}
	}
	for _, args := range [][]string{nil, {}, {"version", "extra"}, {"-version"}} {
		if isVersionRequest(args) {
			t.Fatalf("isVersionRequest(%q) = true", args)
		}
	}
}

func TestPrintVersion(t *testing.T) {
	oldVersion, oldCommit := version, commit
	version, commit = "v0.1.0", "0123456789ab"
	t.Cleanup(func() {
		version, commit = oldVersion, oldCommit
	})

	var output bytes.Buffer
	printVersion(&output)
	if got, want := output.String(), "repoctl v0.1.0 (0123456789ab)\n"; got != want {
		t.Fatalf("printVersion = %q, want %q", got, want)
	}
}
