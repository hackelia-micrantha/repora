package managedartifact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const reviewDiffContextLines = 3

// ReviewDiff renders a deterministic human-review diff over exact line
// segments. It is review output, not an applyable patch. Changed blocks are
// JSON quoted so line-ending-only changes such as CRLF to LF remain visible
// without embedding terminal control bytes.
func ReviewDiff(observedPresent bool, observed, desired []byte) (string, error) {
	if !observedPresent && len(observed) != 0 {
		return "", fmt.Errorf("absent README review input must not include observed content")
	}
	if len(observed) > MaxTextBytes || len(desired) > MaxTextBytes {
		return "", fmt.Errorf("README review input exceeds %d-byte limit", MaxTextBytes)
	}
	if observedPresent {
		if err := validateManagedText("observed README content", string(observed), true); err != nil {
			return "", err
		}
	}
	if err := validateManagedText("desired README content", string(desired), false); err != nil {
		return "", err
	}
	if observedPresent && bytes.Equal(observed, desired) {
		return "", nil
	}

	oldLines := splitReviewLines(observedPresent, string(observed))
	newLines := splitReviewLines(true, string(desired))
	prefix := commonPrefix(oldLines, newLines)
	suffix := commonSuffix(oldLines[prefix:], newLines[prefix:])

	oldChangeEnd := len(oldLines) - suffix
	newChangeEnd := len(newLines) - suffix
	oldStart := maxInt(0, prefix-reviewDiffContextLines)
	newStart := maxInt(0, prefix-reviewDiffContextLines)
	oldEnd := minInt(len(oldLines), oldChangeEnd+reviewDiffContextLines)
	newEnd := minInt(len(newLines), newChangeEnd+reviewDiffContextLines)

	var out strings.Builder
	out.WriteString("--- a/README.md\n")
	out.WriteString("+++ b/README.md\n")
	fmt.Fprintf(&out, "@@ -%d,%d +%d,%d @@\n", unifiedStart(oldStart, oldEnd-oldStart), oldEnd-oldStart, unifiedStart(newStart, newEnd-newStart), newEnd-newStart)

	for _, line := range oldLines[oldStart:prefix] {
		writeReviewLine(&out, ' ', line)
	}
	writeReviewBlock(&out, '-', oldLines[prefix:oldChangeEnd])
	writeReviewBlock(&out, '+', newLines[prefix:newChangeEnd])
	oldSuffixStart := len(oldLines) - suffix
	for _, line := range oldLines[oldSuffixStart:oldEnd] {
		writeReviewLine(&out, ' ', line)
	}

	diff := out.String()
	if len(diff) > MaxDiffBytes {
		return "", fmt.Errorf("README review diff exceeds %d-byte limit", MaxDiffBytes)
	}
	if err := validateReviewDiff(diff); err != nil {
		return "", err
	}
	return diff, nil
}

func splitReviewLines(present bool, value string) []string {
	if !present {
		return nil
	}
	if value == "" {
		return []string{""}
	}
	var lines []string
	for len(value) > 0 {
		index := strings.IndexByte(value, '\n')
		if index < 0 {
			lines = append(lines, value)
			break
		}
		lines = append(lines, value[:index+1])
		value = value[index+1:]
	}
	return lines
}

func commonPrefix(left, right []string) int {
	limit := minInt(len(left), len(right))
	for i := 0; i < limit; i++ {
		if left[i] != right[i] {
			return i
		}
	}
	return limit
}

func commonSuffix(left, right []string) int {
	limit := minInt(len(left), len(right))
	for i := 0; i < limit; i++ {
		if left[len(left)-1-i] != right[len(right)-1-i] {
			return i
		}
	}
	return limit
}

func writeReviewBlock(out *strings.Builder, prefix byte, lines []string) {
	if len(lines) == 0 {
		return
	}
	writeReviewLine(out, prefix, strings.Join(lines, ""))
}

func writeReviewLine(out *strings.Builder, prefix byte, line string) {
	out.WriteByte(prefix)
	out.WriteString(jsonQuote(line))
	out.WriteByte('\n')
}

func jsonQuote(value string) string {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
	return strings.TrimSuffix(buffer.String(), "\n")
}

func unifiedStart(zeroBased, count int) int {
	if count == 0 {
		return zeroBased
	}
	return zeroBased + 1
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
