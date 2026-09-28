package main

import (
	"bytes"
	"os"
	"testing"
)

func TestBootstrapHelpMatchesGolden(t *testing.T) {
	want, err := os.ReadFile("testdata/bootstrap-help.golden.txt")
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"bootstrap", "--help"})
	})
	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if !bytes.Equal(stdout.Bytes(), want) {
		t.Fatalf("bootstrap help changed:\ngot:\n%s\nwant:\n%s", stdout.Bytes(), want)
	}
}
