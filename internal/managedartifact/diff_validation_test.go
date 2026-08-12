package managedartifact

import (
	"strings"
	"testing"
)

func TestReviewDiffValidatesEqualInputBeforeNoOp(t *testing.T) {
	invalid := []byte{0xff}
	_, err := ReviewDiff(true, invalid, invalid)
	if err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
		t.Fatalf("ReviewDiff() error = %v, want invalid equal input rejection", err)
	}
}
