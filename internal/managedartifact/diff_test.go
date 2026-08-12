package managedartifact

import (
	"strings"
	"testing"
)

func TestReviewDiffShowsLineEndingOnlyChange(t *testing.T) {
	got, err := ReviewDiff(true, []byte("# Title\r\n"), []byte("# Title\n"))
	if err != nil {
		t.Fatalf("ReviewDiff() error = %v", err)
	}
	for _, expected := range []string{`-"# Title\r\n"`, `+"# Title\n"`} {
		if !strings.Contains(got, expected) {
			t.Fatalf("ReviewDiff() missing %q:\n%s", expected, got)
		}
	}
	if strings.ContainsRune(got, '\r') {
		t.Fatalf("ReviewDiff() embedded raw CR: %q", got)
	}
}

func TestReviewDiffDistinguishesAbsentFromEmptyFile(t *testing.T) {
	got, err := ReviewDiff(false, nil, []byte{})
	if err != nil {
		t.Fatalf("ReviewDiff() error = %v", err)
	}
	if !strings.Contains(got, `+""`) {
		t.Fatalf("ReviewDiff() = %q, want explicit empty-file review line", got)
	}
}

func TestReviewDiffReturnsEmptyOnlyForEqualPresentFile(t *testing.T) {
	got, err := ReviewDiff(true, []byte("same\n"), []byte("same\n"))
	if err != nil {
		t.Fatalf("ReviewDiff() error = %v", err)
	}
	if got != "" {
		t.Fatalf("ReviewDiff() = %q, want empty diff", got)
	}
}

func TestReviewDiffIsDeterministicWithContext(t *testing.T) {
	old := []byte("one\ntwo\nthree\nfour\nfive\nsix\nseven\n")
	new := []byte("one\ntwo\nthree\nFOUR\nfive\nsix\nseven\n")
	first, err := ReviewDiff(true, old, new)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReviewDiff(true, old, new)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("ReviewDiff() is nondeterministic\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	for _, expected := range []string{` "one\n"`, `-"four\n"`, `+"FOUR\n"`, ` "seven\n"`} {
		if !strings.Contains(first, expected) {
			t.Fatalf("ReviewDiff() missing %q:\n%s", expected, first)
		}
	}
}

func TestReviewDiffRejectsUnsafeObservedText(t *testing.T) {
	_, err := ReviewDiff(true, []byte("safe\u202eunsafe\n"), []byte("safe\n"))
	if err == nil || !strings.Contains(err.Error(), "unsafe display character") {
		t.Fatalf("ReviewDiff() error = %v, want display-safety rejection", err)
	}
}
