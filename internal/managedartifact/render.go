// Package managedartifact owns bounded deterministic rendering for managed repository artifacts.
package managedartifact

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxTextBytes = 1 << 20

var valueKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9._-]*$`)

type RenderData struct {
	RepoID            string
	RepoUID           string
	CanonicalProvider string
	CanonicalPath     string
	Values            map[string]string
}

func RenderREADME(template []byte, data RenderData) ([]byte, error) {
	if len(template) > MaxTextBytes {
		return nil, fmt.Errorf("README template exceeds %d-byte limit", MaxTextBytes)
	}
	if !utf8.Valid(template) {
		return nil, fmt.Errorf("README template must be valid UTF-8")
	}
	templateText, err := normalizeText("README template", string(template))
	if err != nil {
		return nil, err
	}

	replacements, err := renderReplacements(data)
	if err != nil {
		return nil, err
	}

	var output bytes.Buffer
	remaining := templateText
	for len(remaining) > 0 {
		open := strings.Index(remaining, "{{")
		close := strings.Index(remaining, "}}")
		if close >= 0 && (open < 0 || close < open) {
			return nil, fmt.Errorf("README template contains unexpected closing token")
		}
		if open < 0 {
			if err := writeBounded(&output, remaining); err != nil {
				return nil, err
			}
			break
		}
		if err := writeBounded(&output, remaining[:open]); err != nil {
			return nil, err
		}

		afterOpen := remaining[open+2:]
		end := strings.Index(afterOpen, "}}")
		if end < 0 {
			return nil, fmt.Errorf("README template contains unclosed token")
		}
		token := afterOpen[:end]
		if token == "" || strings.ContainsAny(token, "{}\r\n\t ") {
			return nil, fmt.Errorf("README template contains malformed token %q", token)
		}
		replacement, ok := replacements[token]
		if !ok {
			return nil, fmt.Errorf("README template contains unsupported or unresolved token %q", token)
		}
		if err := writeBounded(&output, replacement); err != nil {
			return nil, err
		}
		remaining = afterOpen[end+2:]
	}

	return output.Bytes(), nil
}

func renderReplacements(data RenderData) (map[string]string, error) {
	values := make(map[string]string, 4+len(data.Values))
	builtins := []struct {
		token string
		raw   string
	}{
		{token: "repo.id", raw: data.RepoID},
		{token: "repo.uid", raw: data.RepoUID},
		{token: "canonical.provider", raw: data.CanonicalProvider},
		{token: "canonical.path", raw: data.CanonicalPath},
	}
	for _, builtin := range builtins {
		normalized, err := normalizeText(builtin.token, builtin.raw)
		if err != nil {
			return nil, err
		}
		if normalized != "" {
			values[builtin.token] = normalized
		}
	}

	keys := make([]string, 0, len(data.Values))
	for key := range data.Values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !valueKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("README render value key %q is invalid", key)
		}
		normalized, err := normalizeText("value."+key, data.Values[key])
		if err != nil {
			return nil, err
		}
		values["value."+key] = normalized
	}
	return values, nil
}

func normalizeText(label, value string) (string, error) {
	if err := validateManagedText(label, value, true); err != nil {
		return "", err
	}
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return value, nil
}

func validateManagedText(label, value string, allowCR bool) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s must be valid UTF-8", label)
	}
	if strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("%s must not contain NUL", label)
	}
	for _, r := range value {
		if r == '\r' {
			if allowCR {
				continue
			}
			return fmt.Errorf("%s must use LF line endings", label)
		}
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return fmt.Errorf("%s contains unsupported control character U+%04X", label, r)
		}
		if unicode.Is(unicode.Cf, r) || r == '\u2028' || r == '\u2029' {
			return fmt.Errorf("%s contains unsafe display character U+%04X", label, r)
		}
	}
	return nil
}

func writeBounded(output *bytes.Buffer, value string) error {
	if output.Len()+len(value) > MaxTextBytes {
		return fmt.Errorf("rendered README exceeds %d-byte limit", MaxTextBytes)
	}
	_, _ = output.WriteString(value)
	return nil
}
