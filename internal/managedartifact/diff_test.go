package managedartifact

import (
	"bytes"
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

func TestReviewDiffDistinguishesPresentEmptyFileFromMissing(t *testing.T) {
	got, err := ReviewDiff(true, []byte{}, []byte("text\n"))
	if err != nil {
		t.Fatalf("ReviewDiff() error = %v", err)
	}
	if !strings.Contains(got, `-""`) || !strings.Contains(got, `+"text\n"`) {
		t.Fatalf("ReviewDiff() = %q, want explicit present-empty source", got)
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

func TestReviewDiffRejectsContentForAbsentObservation(t *testing.T) {
	_, err := ReviewDiff(false, []byte("impossible"), []byte("new"))
	if err == nil || !strings.Contains(err.Error(), "must not include observed content") {
		t.Fatalf("ReviewDiff() error = %v, want absent-state rejection", err)
	}
}

func TestReviewDiffIsDeterministicWithContextAndGroupedChanges(t *testing.T) {
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

func TestReviewDiffGroupsLargeChangedBlockWithinBound(t *testing.T) {
	old := bytes.Repeat([]byte{'\t'}, MaxTextBytes)
	new := bytes.Repeat([]byte{'"'}, MaxTextBytes)
	got, err := ReviewDiff(true, old, new)
	if err != nil {
		t.Fatalf("ReviewDiff() worst-case escaping error = %v", err)
	}
	if len(got) > MaxDiffBytes {
		t.Fatalf("ReviewDiff() size = %d, max = %d", len(got), MaxDiffBytes)
	}
	if strings.Count(got, "\n") != 5 {
		t.Fatalf("ReviewDiff() should group changed blocks into two review lines, got %d LF bytes", strings.Count(got, "\n"))
	}
}

func TestReviewDiffRejectsUnsafeObservedText(t *testing.T) {
	_, err := ReviewDiff(true, []byte("safe\u202eunsafe\n"), []byte("safe\n"))
	if err == nil || !strings.Contains(err.Error(), "unsafe display character") {
		t.Fatalf("ReviewDiff() error = %v, want display-safety rejection", err)
	}
}
