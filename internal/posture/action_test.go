package posture

import "testing"

func TestClassifyActionRequiresCompleteImmutableReferences(t *testing.T) {
	sha := "0123456789012345678901234567890123456789"
	digest := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tests := []struct {
		name    string
		uses    string
		pinning string
	}{
		{name: "action sha", uses: "vendor/action@" + sha, pinning: "immutable-sha"},
		{name: "short action sha", uses: "vendor/action@0123456", pinning: "mutable-ref"},
		{name: "docker digest", uses: "docker://ghcr.io/vendor/image@sha256:" + digest, pinning: "immutable-digest"},
		{name: "malformed docker digest", uses: "docker://ghcr.io/vendor/image@sha256:abc", pinning: "mutable-ref"},
		{name: "docker tag", uses: "docker://ghcr.io/vendor/image:v1", pinning: "mutable-ref"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyAction("owner/repo", tt.uses)
			if got.Pinning != tt.pinning {
				t.Fatalf("pinning = %q, want %q for %q", got.Pinning, tt.pinning, tt.uses)
			}
		})
	}
}
