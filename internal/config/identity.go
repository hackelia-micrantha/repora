package config

import (
	"fmt"
	"net/url"
	"strings"
)

// RepositoryPath returns the safe provider-relative identity path for an
// endpoint. Preferred path configuration is returned directly; bounded legacy
// network URLs are reduced to repository path only.
func (e Endpoint) RepositoryPath() (string, error) {
	if path := strings.TrimSpace(e.Path); path != "" {
		return validateRepositoryPath(path)
	}

	raw := strings.TrimSpace(e.URL)
	var path string
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("parse legacy repository URL: %w", err)
		}
		switch strings.ToLower(parsed.Scheme) {
		case "http", "https", "ssh", "git":
		default:
			return "", fmt.Errorf("legacy repository URL scheme %q cannot provide durable provider identity", parsed.Scheme)
		}
		if strings.TrimSpace(parsed.Host) == "" {
			return "", fmt.Errorf("legacy repository URL requires a host")
		}
		path = strings.Trim(parsed.Path, "/")
	} else if at := strings.LastIndex(raw, "@"); at >= 0 {
		hostAndPath := raw[at+1:]
		if colon := strings.Index(hostAndPath, ":"); colon >= 0 {
			if strings.TrimSpace(hostAndPath[:colon]) == "" {
				return "", fmt.Errorf("legacy repository URL requires a host")
			}
			path = strings.Trim(hostAndPath[colon+1:], "/")
		}
	}
	path = strings.TrimSuffix(strings.TrimSpace(path), ".git")
	if path == "" {
		return "", fmt.Errorf("legacy repository URL does not contain a safe provider-relative path")
	}
	return validateRepositoryPath(path)
}

// TargetID returns stable provider/path identity without transport details.
func (e Endpoint) TargetID() (string, error) {
	provider := strings.TrimSpace(e.Provider)
	if provider == "" {
		return "", fmt.Errorf("provider is required")
	}
	path, err := e.RepositoryPath()
	if err != nil {
		return "", err
	}
	return provider + ":" + path, nil
}

func validateRepositoryPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") {
		return "", fmt.Errorf("repository path contains an unsafe segment")
	}
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("repository path must include an owner or namespace")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `\:@?#`) || strings.ContainsAny(part, " \t\r\n") {
			return "", fmt.Errorf("repository path contains an unsafe segment")
		}
	}
	return path, nil
}
