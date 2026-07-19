package status

import (
	"testing"

	"repoctl/internal/config"
)

func TestCheckResolvesProviderPathsBeforeGit(t *testing.T) {
	git := &fakeGitClient{revListOutput: "0\t0\n", revParseShort: "abc123"}
	repo := config.Repo{
		ID:        "payments-api",
		Canonical: config.Endpoint{Provider: "gitlab", Path: "org/platform/payments-api"},
		Mirrors:   []config.Endpoint{{Provider: "github", Path: "org/payments-api"}},
		Mode:      "mirror",
	}

	if _, err := Check(repo, git); err != nil {
		t.Fatal(err)
	}
	if got := git.ensureMirrorCalls[0].url; got != "https://gitlab.com/org/platform/payments-api.git" {
		t.Fatalf("canonical url = %q", got)
	}
	if got := git.configureRemoteCalls[1].url; got != "https://github.com/org/payments-api.git" {
		t.Fatalf("mirror url = %q", got)
	}
}
