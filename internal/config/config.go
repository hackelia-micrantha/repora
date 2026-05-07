package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	Version1              = 1
	DefaultTransportHTTPS = "https"
	DefaultTransportSSH   = "ssh"
	ModeMirror            = "mirror"
)

type Spec struct {
	Version          int                 `json:"version" yaml:"version"`
	DefaultTransport string              `json:"default_transport,omitempty" yaml:"default_transport,omitempty"`
	Providers        map[string]Provider `json:"providers" yaml:"providers"`
	Repos            []Repo              `json:"repos" yaml:"repos"`
}

type Provider struct {
	BaseURLs BaseURLs `json:"base_urls" yaml:"base_urls"`
}

type BaseURLs struct {
	HTTPS string `json:"https" yaml:"https"`
	SSH   string `json:"ssh" yaml:"ssh"`
}

type Repo struct {
	ID        string      `json:"id" yaml:"id"`
	UID       string      `json:"uid" yaml:"uid"`
	Canonical RemoteRef   `json:"canonical" yaml:"canonical"`
	Mirrors   []RemoteRef `json:"mirrors" yaml:"mirrors"`
	Mode      string      `json:"mode" yaml:"mode"`
}

type RemoteRef struct {
	Provider string `json:"provider" yaml:"provider"`
	Path     string `json:"path" yaml:"path"`
	URL      string `json:"-" yaml:"-"`
}

func Load(path string) (Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Spec{}, fmt.Errorf("read config: %w", err)
	}

	spec, err := parse(data)
	if err != nil {
		return Spec{}, err
	}
	if err := validate(&spec); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

func parse(data []byte) (Spec, error) {
	var spec Spec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return Spec{}, fmt.Errorf("parse config: %w", err)
	}
	return spec, nil
}

func validate(spec *Spec) error {
	if spec.Version == 0 {
		return fmt.Errorf("version is required")
	}
	if spec.Version != Version1 {
		return fmt.Errorf("unsupported version %d", spec.Version)
	}

	spec.DefaultTransport = normalizeLower(spec.DefaultTransport)
	if spec.DefaultTransport == "" {
		spec.DefaultTransport = DefaultTransportHTTPS
	}
	if spec.DefaultTransport != DefaultTransportHTTPS && spec.DefaultTransport != DefaultTransportSSH {
		return fmt.Errorf("default_transport must be %q or %q", DefaultTransportHTTPS, DefaultTransportSSH)
	}

	if len(spec.Providers) == 0 {
		return fmt.Errorf("providers is required")
	}

	normalizedProviders := make(map[string]Provider, len(spec.Providers))
	for rawName, provider := range spec.Providers {
		name := normalizeLower(rawName)
		if name == "" {
			return fmt.Errorf("provider name must not be empty")
		}

		provider.BaseURLs.HTTPS = strings.TrimSpace(provider.BaseURLs.HTTPS)
		provider.BaseURLs.SSH = strings.TrimSpace(provider.BaseURLs.SSH)
		if provider.BaseURLs.HTTPS == "" {
			return fmt.Errorf("providers.%s.base_urls.https is required", name)
		}
		if provider.BaseURLs.SSH == "" {
			return fmt.Errorf("providers.%s.base_urls.ssh is required", name)
		}

		normalizedProviders[name] = provider
	}
	spec.Providers = normalizedProviders

	if len(spec.Repos) == 0 {
		return fmt.Errorf("repos must contain at least one entry")
	}

	seenIDs := make(map[string]struct{}, len(spec.Repos))
	seenUIDs := make(map[string]struct{}, len(spec.Repos))
	seenProviderPaths := make(map[string]string)

	for i := range spec.Repos {
		repo := &spec.Repos[i]
		repo.ID = strings.TrimSpace(repo.ID)
		repo.UID = strings.TrimSpace(repo.UID)
		repo.Mode = normalizeLower(repo.Mode)

		if repo.ID == "" {
			return fmt.Errorf("repos[%d].id is required", i)
		}
		if repo.UID == "" {
			return fmt.Errorf("repos[%d].uid is required", i)
		}
		if _, ok := seenIDs[repo.ID]; ok {
			return fmt.Errorf("repos[%d].id: duplicate value %q", i, repo.ID)
		}
		if _, ok := seenUIDs[repo.UID]; ok {
			return fmt.Errorf("repos[%d].uid: duplicate value %q", i, repo.UID)
		}
		seenIDs[repo.ID] = struct{}{}
		seenUIDs[repo.UID] = struct{}{}

		if repo.Mode == "" {
			repo.Mode = ModeMirror
		}
		if repo.Mode != ModeMirror {
			return fmt.Errorf("repos[%d].mode: unsupported value %q", i, repo.Mode)
		}

		if err := validateRemoteRef(spec.Providers, spec.DefaultTransport, &repo.Canonical, fmt.Sprintf("repos[%d].canonical", i)); err != nil {
			return err
		}
		if err := recordProviderPathUnique(seenProviderPaths, repo.Canonical.Provider, repo.Canonical.Path, repo.UID); err != nil {
			return fmt.Errorf("repos[%d].canonical.path: %w", i, err)
		}

		if len(repo.Mirrors) == 0 {
			return fmt.Errorf("repos[%d].mirrors must contain at least one entry", i)
		}

		mirrorProviders := make(map[string]struct{}, len(repo.Mirrors))
		for j := range repo.Mirrors {
			ref := &repo.Mirrors[j]
			fieldPath := fmt.Sprintf("repos[%d].mirrors[%d]", i, j)
			if err := validateRemoteRef(spec.Providers, spec.DefaultTransport, ref, fieldPath); err != nil {
				return err
			}
			if ref.Provider == repo.Canonical.Provider {
				return fmt.Errorf("%s.provider: mirror provider must not equal canonical provider %q", fieldPath, repo.Canonical.Provider)
			}
			if _, ok := mirrorProviders[ref.Provider]; ok {
				return fmt.Errorf("%s.provider: duplicate mirror provider %q", fieldPath, ref.Provider)
			}
			mirrorProviders[ref.Provider] = struct{}{}
			if err := recordProviderPathUnique(seenProviderPaths, ref.Provider, ref.Path, repo.UID); err != nil {
				return fmt.Errorf("%s.path: %w", fieldPath, err)
			}
		}
	}

	return nil
}

func validateRemoteRef(providers map[string]Provider, transport string, ref *RemoteRef, fieldPath string) error {
	ref.Provider = normalizeLower(ref.Provider)
	ref.Path = normalizePath(ref.Path)

	if ref.Provider == "" {
		return fmt.Errorf("%s.provider is required", fieldPath)
	}
	provider, ok := providers[ref.Provider]
	if !ok {
		return fmt.Errorf("%s.provider: unknown provider %q", fieldPath, ref.Provider)
	}
	if ref.Path == "" {
		return fmt.Errorf("%s.path is required", fieldPath)
	}
	if err := validatePath(ref.Path); err != nil {
		return fmt.Errorf("%s.path: %w", fieldPath, err)
	}
	url, err := DeriveURL(provider, ref.Path, transport)
	if err != nil {
		return fmt.Errorf("%s: derive url: %w", fieldPath, err)
	}
	ref.URL = url
	return nil
}

func normalizeLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func normalizePath(path string) string {
	return strings.TrimSpace(path)
}

func validatePath(path string) error {
	switch {
	case path == "":
		return fmt.Errorf("must not be empty")
	case strings.HasPrefix(path, "/"):
		return fmt.Errorf("must not start with '/'")
	case strings.HasSuffix(path, "/"):
		return fmt.Errorf("must not end with '/'")
	case strings.Contains(path, ".git"):
		return fmt.Errorf("must not contain '.git'")
	case strings.Contains(path, "//"):
		return fmt.Errorf("must not contain empty path segments")
	case strings.Contains(path, "://"):
		return fmt.Errorf("must not contain a URL scheme")
	case strings.Contains(path, "@") || strings.Contains(path, ":"):
		return fmt.Errorf("must not contain remote-style prefixes")
	case strings.Contains(path, "?") || strings.Contains(path, "#"):
		return fmt.Errorf("must not contain query or fragment components")
	}
	return nil
}

func recordProviderPathUnique(seen map[string]string, provider, path, uid string) error {
	key := provider + "::" + path
	if prior, ok := seen[key]; ok && prior != uid {
		return fmt.Errorf("duplicate provider/path %q on provider %q already assigned to %q", path, provider, prior)
	}
	seen[key] = uid
	return nil
}

func DeriveURL(provider Provider, path, transport string) (string, error) {
	switch normalizeLower(transport) {
	case DefaultTransportHTTPS:
		base := strings.TrimRight(strings.TrimSpace(provider.BaseURLs.HTTPS), "/")
		if base == "" {
			return "", fmt.Errorf("provider https base url is empty")
		}
		return base + "/" + path + ".git", nil
	case DefaultTransportSSH:
		base := strings.TrimSpace(provider.BaseURLs.SSH)
		if base == "" {
			return "", fmt.Errorf("provider ssh base url is empty")
		}
		return base + path + ".git", nil
	default:
		return "", fmt.Errorf("unsupported transport %q", transport)
	}
}
