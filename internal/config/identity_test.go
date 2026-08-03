package config

import (
	"strings"
	"testing"
)

func TestEndpointRepositoryPathUsesDeclarativePath(t *testing.T) {
	got, err := (Endpoint{Provider: "github", Path: "org/payments-api"}).RepositoryPath()
	if err != nil {
		t.Fatalf("RepositoryPath returned error: %v", err)
	}
	if got != "org/payments-api" {
		t.Fatalf("path = %q, want org/payments-api", got)
	}
}

func TestEndpointRepositoryPathDerivesSafeLegacyPath(t *testing.T) {
	for _, raw := range []string{
		"git@github.com:org/payments-api.git",
		"https://github.com/org/payments-api.git?token=ignored#fragment",
	} {
		got, err := (Endpoint{Provider: "github", URL: raw}).RepositoryPath()
		if err != nil {
			t.Fatalf("RepositoryPath(%q) returned error: %v", raw, err)
		}
		if got != "org/payments-api" {
			t.Fatalf("RepositoryPath(%q) = %q, want org/payments-api", raw, got)
		}
	}
}

func TestEndpointRepositoryPathRejectsUnsafeIdentity(t *testing.T) {
	for _, endpoint := range []Endpoint{
		{Provider: "github", Path: "/tmp/payments-api"},
		{Provider: "github", Path: "../payments-api"},
		{Provider: "github", URL: "file:///tmp/payments-api.git"},
	} {
		_, err := endpoint.RepositoryPath()
		if err == nil || !strings.Contains(err.Error(), "path") {
			t.Fatalf("endpoint %#v error = %v, want unsafe path rejection", endpoint, err)
		}
	}
}

func TestEndpointTargetIDExcludesTransport(t *testing.T) {
	got, err := (Endpoint{Provider: "github", URL: "https://github.com/org/payments-api.git?token=secret"}).TargetID()
	if err != nil {
		t.Fatalf("TargetID returned error: %v", err)
	}
	if got != "github:org/payments-api" || strings.Contains(got, "token") || strings.Contains(got, "github.com") {
		t.Fatalf("target ID = %q, want provider-relative identity only", got)
	}
}
