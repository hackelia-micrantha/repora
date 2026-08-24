package transport

import (
	"strings"
	"testing"

	"repoctl/internal/config"
)

func TestResolveProviderPath(t *testing.T) {
	t.Parallel()

	endpoint := config.Endpoint{Provider: "gitlab", Path: "group/subgroup/repo"}

	https, err := DefaultResolver(HTTPS).Resolve(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	if https.URL != "https://gitlab.com/group/subgroup/repo.git" {
		t.Fatalf("https url = %q", https.URL)
	}

	ssh, err := DefaultResolver(SSH).Resolve(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	if ssh.URL != "git@gitlab.com:group/subgroup/repo.git" {
		t.Fatalf("ssh url = %q", ssh.URL)
	}
}

func TestResolveBitbucketCloudMirror(t *testing.T) {
	t.Parallel()

	resolved, err := DefaultResolver(HTTPS).Resolve(config.Endpoint{
		Provider: "bitbucket",
		Path:     "micrantha/anthesis",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Provider != "bitbucket" || resolved.Path != "micrantha/anthesis" {
		t.Fatalf("identity = %#v", resolved)
	}
	if resolved.URL != "https://bitbucket.org/micrantha/anthesis.git" {
		t.Fatalf("https url = %q", resolved.URL)
	}
	if strings.Contains(resolved.URL, "@") {
		t.Fatalf("resolved URL contains credentials: %q", resolved.URL)
	}
}

func TestResolveBitbucketFailsClosed(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"workspace/group/repo",
		"workspace/repo?token=secret",
		"workspace/repo#fragment",
		`workspace/repo\other`,
		"workspace/repo:other",
		"/workspace/repo",
		"workspace/repo/",
		"workspace/repo.git",
		"work space/repo",
		"workspace/repo name",
		"work\tspace/repo",
		"workspace/repo\nname",
		"work\u00a0space/repo",
	} {
		_, err := DefaultResolver(HTTPS).Resolve(config.Endpoint{Provider: "bitbucket", Path: path})
		if err == nil {
			t.Fatalf("Resolve(%q) returned nil error", path)
		}
	}

	_, err := DefaultResolver(HTTPS).Resolve(config.Endpoint{
		Provider: "bitbucket",
		URL:      "https://bitbucket.org/workspace/repo.git",
	})
	if err == nil || !strings.Contains(err.Error(), "provider/path identity is required") {
		t.Fatalf("legacy URL error = %v, want provider/path-only rejection", err)
	}

	_, err = DefaultResolver(SSH).Resolve(config.Endpoint{Provider: "bitbucket", Path: "workspace/repo"})
	if err == nil || !strings.Contains(err.Error(), "no ssh base") {
		t.Fatalf("ssh error = %v, want unsupported SSH transport", err)
	}
}

func TestResolveFailsClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resolver Resolver
		endpoint config.Endpoint
		contains string
	}{
		{"unknown provider", DefaultResolver(HTTPS), config.Endpoint{Provider: "other", Path: "org/repo"}, "unknown provider"},
		{"invalid path", DefaultResolver(HTTPS), config.Endpoint{Provider: "github", Path: "../repo"}, "provider-relative"},
		{"missing base", Resolver{Transport: SSH, Providers: map[string]Provider{"github": {HTTPSBase: "https://github.com"}}}, config.Endpoint{Provider: "github", Path: "org/repo"}, "no ssh base"},
		{"ambiguous endpoint", DefaultResolver(HTTPS), config.Endpoint{Provider: "github", Path: "org/repo", URL: "https://github.com/org/repo.git"}, "both path and legacy url"},
		{"credential url", DefaultResolver(HTTPS), config.Endpoint{Provider: "github", URL: "https://token@github.com/org/repo.git"}, "credential-bearing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.resolver.Resolve(tt.endpoint)
			if err == nil || !strings.Contains(err.Error(), tt.contains) {
				t.Fatalf("error = %v, want containing %q", err, tt.contains)
			}
		})
	}
}

func TestResolveLegacyURL(t *testing.T) {
	t.Parallel()

	resolved, err := DefaultResolver(HTTPS).Resolve(config.Endpoint{
		Provider: "github",
		URL:      "git@github.com:org/repo.git",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.URL != "git@github.com:org/repo.git" || resolved.Transport != "legacy" {
		t.Fatalf("resolved = %#v", resolved)
	}
}
