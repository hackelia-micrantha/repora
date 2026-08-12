// Package managedartifact contains bounded deterministic rendering for managed repository artifacts.
package managedartifact

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const MaxTextBytes = 256 * 1024

var valueKeyPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

type RenderContext struct {
	RepoID            string
	RepoUID           string
	CanonicalProvider string
	CanonicalPath     string
	Values            map[string]string
}

func RenderREADME(template []byte, context RenderContext) ([]byte, error) {
	if len(template) > MaxTextBytes {
		return nil, fmt.Errorf("README template exceeds %d-byte limit", MaxTextBytes)
	}
	if !utf8.Valid(template) || strings.IndexByte(string(template), 0) >= 0 {
		return nil, fmt.Errorf("README template must be valid UTF-8 text without NUL bytes")
	}

	input := normalizeLF(string(template))
	values := map[string]string{
		"repo.id":            context.RepoID,
		"repo.uid":           context.RepoUID,
		"canonical.provider": context.CanonicalProvider,
		"canonical.path":     context.CanonicalPath,
	}
	for key, value := range values {
		normalized, err := normalizeReplacement(key, value)
		if err != nil {
			return nil, err
		}
		values[key] = normalized
	}
	configured := make(map[string]string, len(context.Values))
	for key, value := range context.Values {
		if !valueKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("invalid README value key %q", key)
		}
		normalized, err := normalizeReplacement("value."+key, value)
		if err != nil {
			return nil, err
		}
		configured[key] = normalized
	}

	var output strings.Builder
	for len(input) > 0 {
		open := strings.Index(input, "{{")
		closeOnly := strings.Index(input, "}}")
		if open < 0 {
			if closeOnly >= 0 {
				return nil, fmt.Errorf("malformed README placeholder")
			}
			if err := appendBounded(&output, input); err != nil {
				return nil, err
			}
			break
		}
		if closeOnly >= 0 && closeOnly < open {
			return nil, fmt.Errorf("malformed README placeholder")
		}
		if err := appendBounded(&output, input[:open]); err != nil {
			return nil, err
		}
		input = input[open+2:]
		close := strings.Index(input, "}}")
		if close < 0 || strings.Contains(input[:close], "{{") {
			return nil, fmt.Errorf("malformed README placeholder")
		}
		token := input[:close]
		input = input[close+2:]

		replacement, err := resolveToken(token, values, configured)
		if err != nil {
			return nil, err
		}
		if err := appendBounded(&output, replacement); err != nil {
			return nil, err
		}
	}

	return []byte(output.String()), nil
}

func resolveToken(token string, fixed, configured map[string]string) (string, error) {
	if replacement, ok := fixed[token]; ok {
		return replacement, nil
	}
	if strings.HasPrefix(token, "value.") {
		key := strings.TrimPrefix(token, "value.")
		if !valueKeyPattern.MatchString(key) {
			return "", fmt.Errorf("malformed README placeholder %q", token)
		}
		replacement, ok := configured[key]
		if !ok {
			return "", fmt.Errorf("README placeholder %q references an unconfigured value", token)
		}
		return replacement, nil
	}
	return "", fmt.Errorf("unknown README placeholder %q", token)
}

func normalizeReplacement(name, value string) (string, error) {
	if !utf8.ValidString(value) || strings.IndexByte(value, 0) >= 0 {
		return "", fmt.Errorf("README replacement %q must be valid UTF-8 text without NUL bytes", name)
	}
	return normalizeLF(value), nil
}

func normalizeLF(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.ReplaceAll(value, "\r", "\n")
}

func appendBounded(output *strings.Builder, value string) error {
	if output.Len()+len(value) > MaxTextBytes {
		return fmt.Errorf("rendered README exceeds %d-byte limit", MaxTextBytes)
	}
	output.WriteString(value)
	return nil
}
