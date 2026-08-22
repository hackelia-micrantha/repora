package posture

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"repoctl/internal/config"
)

func TestDefaultMirrorProviderReaderObservesGitHubVisibilityAndPushPermission(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/repos/acme/example" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"default_branch":"main","visibility":"private","permissions":{"push":true}}`))
	}))
	t.Cleanup(server.Close)

	reader := NewHTTPGitHubReader("test-token")
	reader.BaseURL = server.URL
	facts, observation, err := (DefaultMirrorProviderReader{GitHub: reader}).Repository(
		context.Background(),
		config.Endpoint{Provider: "github", Path: "acme/example"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !observation.Available {
		t.Fatalf("observation = %#v", observation)
	}
	if facts.DefaultBranch != "main" || facts.Visibility != "private" {
		t.Fatalf("repository facts = %#v", facts)
	}
	if facts.PushPermission == nil || !*facts.PushPermission {
		t.Fatalf("push permission = %#v", facts.PushPermission)
	}
}

func TestDefaultMirrorProviderReaderLeavesOmittedPermissionsUnknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"default_branch":"main","visibility":"public"}`))
	}))
	t.Cleanup(server.Close)

	reader := NewHTTPGitHubReader("")
	reader.BaseURL = server.URL
	facts, observation, err := (DefaultMirrorProviderReader{GitHub: reader}).Repository(
		context.Background(),
		config.Endpoint{Provider: "github", Path: "acme/example"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !observation.Available || facts.PushPermission != nil {
		t.Fatalf("facts=%#v observation=%#v", facts, observation)
	}
}

func TestDefaultMirrorProviderReaderMarksUnsupportedProviderUnavailable(t *testing.T) {
	facts, observation, err := (DefaultMirrorProviderReader{}).Repository(
		context.Background(),
		config.Endpoint{Provider: "gitlab", Path: "acme/example"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if observation.Available || facts != (MirrorProviderRepository{}) {
		t.Fatalf("facts=%#v observation=%#v", facts, observation)
	}
	if observation.Evidence.Source != "provider.unsupported" {
		t.Fatalf("evidence = %#v", observation.Evidence)
	}
}
