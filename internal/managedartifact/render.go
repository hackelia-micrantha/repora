// Package managedartifact owns bounded deterministic rendering for managed repository artifacts.
package managedartifact

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
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
	values := map[string]string{
		"repo.id":            data.RepoID,
		"repo.uid":           data.RepoUID,
		"canonical.provider": data.CanonicalProvider,
		"canonical.path":     data.CanonicalPath,
	}
	for token, raw := range values {
		normalized, err := normalizeText(token, raw)
		if err != nil {
			return nil, err
		}
		values[token] = normalized
	}
	for key, raw := range data.Values {
		if !valueKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("README render value key %q is invalid", key)
		}
		normalized, err := normalizeText("value."+key, raw)
		if err != nil {
			return nil, err
		}
		values["value."+key] = normalized
	}
	return values, nil
}

func normalizeText(label, value string) (string, error) {
	if !utf8.ValidString(value) {
		return "", fmt.Errorf("%s must be valid UTF-8", label)
	}
	if strings.ContainsRune(value, '\x00') {
		return "", fmt.Errorf("%s must not contain NUL", label)
	}
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return value, nil
}

func writeBounded(output *bytes.Buffer, value string) error {
	if output.Len()+len(value) > MaxTextBytes {
		return fmt.Errorf("rendered README exceeds %d-byte limit", MaxTextBytes)
	}
	_, _ = output.WriteString(value)
	return nil
}
