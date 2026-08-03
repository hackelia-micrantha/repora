package status

import (
	"errors"
	"strings"
	"testing"

	"repoctl/internal/config"
)

type multiGit struct {
	configured []string
	fetched    []string
	heads      []string
	counts     map[string]string
	commits    map[string]string
	fetchErr   map[string]error
}

func (f *multiGit) EnsureMirror(string, string) error { return nil }
func (f *multiGit) ConfigureRemote(_ string, name, _ string) error {
	f.configured = append(f.configured, name)
	return nil
}
func (f *multiGit) Fetch(_ string, name string) error {
	f.fetched = append(f.fetched, name)
	return f.fetchErr[name]
}
func (f *multiGit) SetRemoteHead(_ string, name string) error {
	f.heads = append(f.heads, name)
	return nil
}
func (f *multiGit) RevListLeftRightCount(_, _, right string) (string, error) {
	return f.counts[right], nil
}
func (f *multiGit) RevParseShort(_ string, rev string) (string, error) {
	return f.commits[rev], nil
}

func TestCheckAllSharesCanonicalAndObservesMirrorsInOrder(t *testing.T) {
	git := &multiGit{
		counts: map[string]string{
			"mirror-0/HEAD": "0\t0\n",
			"mirror-1/HEAD": "3\t0\n",
		},
		commits: map[string]string{
			"canonical/HEAD": "canon123",
			"mirror-0/HEAD":  "first123",
			"mirror-1/HEAD":  "second123",
		},
		fetchErr: map[string]error{},
	}
	repo := config.Repo{
		ID:        "payments-api",
		UID:       "repo.org.payments-api",
		Canonical: config.Endpoint{Provider: "gitlab", Path: "org/payments-api"},
		Mirrors: []config.Endpoint{
			{Provider: "github", Path: "org/payments-api"},
			{Provider: "gitlab", Path: "backup/payments-api"},
		},
	}

	got, err := CheckAll(repo, git)
	if err != nil {
		t.Fatalf("CheckAll returned error: %v", err)
	}
	if got.Canonical.Commit != "canon123" || len(got.Mirrors) != 2 {
		t.Fatalf("result = %#v", got)
	}
	if got.Mirrors[0].Target != "github:org/payments-api" || got.Mirrors[0].State != StateEqual {
		t.Fatalf("first mirror = %#v", got.Mirrors[0])
	}
	if got.Mirrors[1].Target != "gitlab:backup/payments-api" || got.Mirrors[1].State != StateBehind || got.Mirrors[1].Behind != 3 {
		t.Fatalf("second mirror = %#v", got.Mirrors[1])
	}
	if strings.Join(git.configured, ",") != "canonical,mirror-0,mirror-1" {
		t.Fatalf("configured remotes = %#v", git.configured)
	}
}

func TestCheckAllPreservesMirrorFailureAndLaterSuccess(t *testing.T) {
	git := &multiGit{
		counts: map[string]string{"mirror-1/HEAD": "0\t2\n"},
		commits: map[string]string{
			"canonical/HEAD": "canon123",
			"mirror-1/HEAD":  "second123",
		},
		fetchErr: map[string]error{"mirror-0": errors.New("remote unavailable")},
	}
	repo := config.Repo{
		ID:        "payments-api",
		UID:       "repo.org.payments-api",
		Canonical: config.Endpoint{Provider: "gitlab", Path: "org/payments-api"},
		Mirrors: []config.Endpoint{
			{Provider: "github", Path: "org/payments-api"},
			{Provider: "gitlab", Path: "backup/payments-api"},
		},
	}

	got, err := CheckAll(repo, git)
	if err == nil || !strings.Contains(err.Error(), "github:org/payments-api") {
		t.Fatalf("error = %v, want target-specific failure", err)
	}
	if len(got.Mirrors) != 2 || got.Mirrors[0].State != StateError || !strings.Contains(got.Mirrors[0].Error, "remote unavailable") {
		t.Fatalf("first mirror = %#v", got.Mirrors[0])
	}
	if got.Mirrors[1].State != StateAhead || got.Mirrors[1].Ahead != 2 {
		t.Fatalf("later mirror = %#v, want preserved success", got.Mirrors[1])
	}
}

func TestCheckRejectsMultipleMirrorsForReconciliation(t *testing.T) {
	repo := testRepo()
	repo.Mirrors = append(repo.Mirrors, config.Endpoint{Provider: "gitlab", Path: "backup/payments-api"})
	if _, err := Check(repo, &multiGit{}); err == nil || !strings.Contains(err.Error(), "exactly one mirror") {
		t.Fatalf("error = %v, want reconciliation gate", err)
	}
}

func TestSafeLegacyPathExcludesTransportDetails(t *testing.T) {
	for _, raw := range []string{
		"git@github.com:org/payments-api.git",
		"https://github.com/org/payments-api.git?token=ignored#fragment",
	} {
		got, err := safeLegacyPath(raw)
		if err != nil {
			t.Fatalf("safeLegacyPath(%q): %v", raw, err)
		}
		if got != "org/payments-api" {
			t.Fatalf("safeLegacyPath(%q) = %q", raw, got)
		}
	}
}
