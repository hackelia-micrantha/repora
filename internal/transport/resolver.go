package transport

import (
	"fmt"
	"net/url"
	"strings"

	"repoctl/internal/config"
)

type Kind string

const (
	HTTPS Kind = "https"
	SSH   Kind = "ssh"
)

type Provider struct {
	HTTPSBase string
	SSHBase   string
}

type ResolvedRemote struct {
	Provider  string
	Path      string
	URL       string
	Transport Kind
}

type Resolver struct {
	Providers map[string]Provider
	Transport Kind
}

func DefaultResolver(kind Kind) Resolver {
	return Resolver{
		Transport: kind,
		Providers: map[string]Provider{
			"bitbucket": {HTTPSBase: "https://bitbucket.org"},
			"github":    {HTTPSBase: "https://github.com", SSHBase: "git@github.com:"},
			"gitlab":    {HTTPSBase: "https://gitlab.com", SSHBase: "git@gitlab.com:"},
		},
	}
}

func (r Resolver) Resolve(endpoint config.Endpoint) (ResolvedRemote, error) {
	provider := strings.TrimSpace(endpoint.Provider)
	path := strings.Trim(strings.TrimSpace(endpoint.Path), "/")

	if path == "" {
		return r.resolveLegacyURL(provider, endpoint.URL)
	}
	if strings.TrimSpace(endpoint.URL) != "" {
		return ResolvedRemote{}, fmt.Errorf("endpoint %s/%s defines both path and legacy url", provider, path)
	}
	if err := validatePath(path); err != nil {
		return ResolvedRemote{}, fmt.Errorf("resolve %s endpoint path %q: %w", provider, path, err)
	}
	if provider == "bitbucket" {
		if err := validateBitbucketPath(path); err != nil {
			return ResolvedRemote{}, fmt.Errorf("resolve bitbucket endpoint path %q: %w", path, err)
		}
	}

	definition, ok := r.Providers[provider]
	if !ok {
		return ResolvedRemote{}, fmt.Errorf("resolve endpoint: unknown provider %q", provider)
	}

	var resolvedURL string
	switch r.Transport {
	case HTTPS:
		if definition.HTTPSBase == "" {
			return ResolvedRemote{}, fmt.Errorf("resolve %s/%s: provider has no https base", provider, path)
		}
		resolvedURL = strings.TrimRight(definition.HTTPSBase, "/") + "/" + ensureGitSuffix(path)
	case SSH:
		if definition.SSHBase == "" {
			return ResolvedRemote{}, fmt.Errorf("resolve %s/%s: provider has no ssh base", provider, path)
		}
		resolvedURL = definition.SSHBase + ensureGitSuffix(path)
	default:
		return ResolvedRemote{}, fmt.Errorf("resolve %s/%s: unsupported transport %q", provider, path, r.Transport)
	}

	return ResolvedRemote{Provider: provider, Path: path, URL: resolvedURL, Transport: r.Transport}, nil
}

func (r Resolver) resolveLegacyURL(provider, raw string) (ResolvedRemote, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ResolvedRemote{}, fmt.Errorf("resolve %s endpoint: path is required", provider)
	}
	if containsCredentials(raw) {
		return ResolvedRemote{}, fmt.Errorf("resolve %s legacy endpoint: credential-bearing urls are not allowed", provider)
	}
	return ResolvedRemote{Provider: provider, URL: raw, Transport: "legacy"}, nil
}

func validatePath(path string) error {
	if strings.Contains(path, "://") || strings.Contains(path, "@") || strings.HasPrefix(path, ".") {
		return fmt.Errorf("must be a provider-relative repository path")
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("contains an invalid path segment")
		}
	}
	if len(strings.Split(path, "/")) < 2 {
		return fmt.Errorf("must include an owner or namespace")
	}
	return nil
}

func validateBitbucketPath(path string) error {
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return fmt.Errorf("must be workspace/repository")
	}
	for _, part := range parts {
		if strings.TrimSpace(part) != part || strings.ContainsAny(part, `\:@?#`) {
			return fmt.Errorf("contains an unsafe workspace or repository segment")
		}
	}
	return nil
}

func containsCredentials(raw string) bool {
	if parsed, err := url.Parse(raw); err == nil && parsed.User != nil {
		return true
	}
	if strings.HasPrefix(raw, "git@") {
		return false
	}
	return strings.Contains(raw, "@")
}

func ensureGitSuffix(path string) string {
	if strings.HasSuffix(path, ".git") {
		return path
	}
	return path + ".git"
}
