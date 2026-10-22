// Package governedmirror performs bounded, read-only validation of a reviewed
// mirror plan before any future Gateway or Anthesis integration. It grants no
// authorization and cannot execute Git.
package governedmirror

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"

	"repoctl/internal/planartifact"
)

const (
	// MaxArtifactBytes bounds parsing of caller-supplied reviewed plan bytes.
	MaxArtifactBytes = 4 << 20
	maxJSONDepth     = 64
)

// ReviewedPlan exposes only non-authorizing metadata from the validated
// bytes. Deliberately do not expose a mutable parsed action alongside the
// digest: changing a parsed action without changing the digest would break
// the reviewed-bytes-to-effect binding required by any future effector.
type ReviewedPlan struct {
	Digest        string
	RequiresForce bool
}

// ParseReviewedPlan is a transport-neutral, non-authorizing preparation
// boundary. The initial governed pilot is deliberately restricted to exactly
// one repository and one action. Multi-target/multi-repository authority must
// be introduced only under a separately reviewed effect-time freshness design.
//
// destructiveIntent is an additional caller intent signal, never a substitute
// for the reviewed plan's force flag or the existing executor's --force and
// force-with-lease checks.
func ParseReviewedPlan(data []byte, destructiveIntent bool) (ReviewedPlan, error) {
	if len(data) == 0 || len(data) > MaxArtifactBytes {
		return ReviewedPlan{}, fmt.Errorf("reviewed plan size must be between 1 and %d bytes", MaxArtifactBytes)
	}
	if !utf8.Valid(data) {
		return ReviewedPlan{}, fmt.Errorf("reviewed plan is not valid UTF-8")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := rejectDuplicateKeys(dec, 0); err != nil {
		return ReviewedPlan{}, fmt.Errorf("reviewed plan JSON: %w", err)
	}
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return ReviewedPlan{}, fmt.Errorf("reviewed plan JSON contains trailing data")
		}
		return ReviewedPlan{}, fmt.Errorf("reviewed plan JSON trailing data: %w", err)
	}
	artifact, err := planartifact.Parse(data)
	if err != nil {
		return ReviewedPlan{}, fmt.Errorf("reviewed plan: %w", err)
	}
	if artifact.Version != planartifact.Version {
		return ReviewedPlan{}, fmt.Errorf("governed preparation requires v2 plan artifact")
	}
	if len(artifact.Repositories) != 1 || len(artifact.Repositories[0].Actions) != 1 {
		return ReviewedPlan{}, fmt.Errorf("governed preparation requires exactly one repository and one mirror action")
	}
	if artifact.Repositories[0].Actions[0].Force && !destructiveIntent {
		return ReviewedPlan{}, fmt.Errorf("reviewed forced action requires separate explicit destructive intent")
	}

	digest := sha256.Sum256(data)
	return ReviewedPlan{Digest: hex.EncodeToString(digest[:]), RequiresForce: artifact.Repositories[0].Actions[0].Force}, nil
}

// rejectDuplicateKeys walks the complete JSON token stream. encoding/json's
// ordinary struct decoder accepts duplicate keys (last value wins), so this
// check must precede parsing into the authoritative plan type. Keys are
// compared after JSON escape decoding, including within nested objects.
func rejectDuplicateKeys(dec *json.Decoder, depth int) error {
	if depth > maxJSONDepth {
		return fmt.Errorf("JSON nesting exceeds %d levels", maxJSONDepth)
	}
	token, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		for dec.More() {
			t, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := t.(string)
			if !ok {
				return fmt.Errorf("invalid object key")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate object key %q", key)
			}
			seen[key] = struct{}{}
			if err := rejectDuplicateKeys(dec, depth+1); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil {
			return err
		}
		if end != json.Delim('}') {
			return fmt.Errorf("invalid object terminator")
		}
	case '[':
		for dec.More() {
			if err := rejectDuplicateKeys(dec, depth+1); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil {
			return err
		}
		if end != json.Delim(']') {
			return fmt.Errorf("invalid array terminator")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
	return nil
}
