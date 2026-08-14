package posture

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPGitHubReaderMapsPermissionDenialToUnavailableEvidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("request method = %s, want GET", r.Method)
		}
		http.Error(w, "forbidden provider detail", http.StatusForbidden)
	}))
	defer server.Close()

	reader := NewHTTPGitHubReader("secret-token")
	reader.BaseURL = server.URL
	_, observation, err := reader.Repository(context.Background(), "acme/project")
	if err != nil {
		t.Fatalf("repository read returned operational error: %v", err)
	}
	if observation.Available {
		t.Fatal("HTTP 403 evidence reported as available")
	}
	if observation.Evidence.Source != "github.repository" {
		t.Fatalf("evidence source = %q", observation.Evidence.Source)
	}
	if !strings.Contains(observation.Evidence.Detail, "HTTP 403") {
		t.Fatalf("evidence detail = %q", observation.Evidence.Detail)
	}
	if strings.Contains(observation.Evidence.Detail, "secret-token") || strings.Contains(observation.Evidence.Detail, "forbidden provider detail") {
		t.Fatalf("sensitive/provider body leaked into evidence detail: %q", observation.Evidence.Detail)
	}
}
