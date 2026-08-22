package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/posture"
)

func validMirrorInventory() posture.MirrorInventory {
	return posture.MirrorInventory{
		Kind:    posture.MirrorInventoryKind,
		Version: posture.MirrorInventoryVersion,
		Repos: []posture.MirrorRepositoryFacts{
			{
				ID:        "example",
				UID:       "repo.example",
				Mode:      posture.Observed("mirror"),
				Direction: posture.Observed("canonical_to_mirror"),
				Canonical: posture.MirrorCanonicalFacts{
					Identity:                   posture.MirrorEndpointIdentity{Provider: "gitlab", Path: "acme/example"},
					DefaultBranch:              posture.Observed("main"),
					Commit:                     posture.Observed("abc1234"),
					Visibility:                 posture.Unavailable[string](),
					CurrentActorPushPermission: posture.Unavailable[bool](),
				},
				Mirrors: []posture.MirrorTargetFacts{
					{
						Identity:                   posture.MirrorEndpointIdentity{Provider: "github", Path: "acme/example"},
						CacheRemote:                posture.Observed("mirror-0"),
						DefaultBranch:              posture.Observed("main"),
						DefaultBranchDrift:         posture.Observed(false),
						Commit:                     posture.Observed("abc1234"),
						Divergence:                 posture.Observed("EQUAL"),
						Ahead:                      posture.Observed(0),
						Behind:                     posture.Observed(0),
						Visibility:                 posture.Unknown[string](),
						CurrentActorPushPermission: posture.Unknown[bool](),
						TagDrift:                   posture.Unknown[bool](),
						ReleaseDrift:               posture.Unknown[bool](),
					},
				},
				Evidence: []posture.Evidence{},
			},
		},
		Evidence: []posture.Evidence{},
	}
}

func TestPostureMirrorsCommandLoadsConfigAndEmitsVersionedJSON(t *testing.T) {
	old := collectMirrorPosture
	t.Cleanup(func() { collectMirrorPosture = old })
	t.Setenv("GITHUB_TOKEN", "mirror-token")

	dir := t.TempDir()
	configPath := filepath.Join(dir, "repora.yaml")
	data := []byte("repos:\n  - id: example\n    uid: repo.example\n    canonical:\n      provider: gitlab\n      path: acme/example\n    mirrors:\n      - provider: github\n        path: acme/example\n    mode: mirror\n")
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	var gotSpec config.Spec
	var gotToken string
	collectMirrorPosture = func(_ context.Context, spec config.Spec, token string) (posture.MirrorInventory, error) {
		gotSpec, gotToken = spec, token
		return validMirrorInventory(), nil
	}

	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"posture", "mirrors", "-f", configPath})
	})
	if code != 0 {
		t.Fatalf("run returned %d", code)
	}
	if len(gotSpec.Repos) != 1 || gotSpec.Repos[0].DurableID() != "repo.example" || gotToken != "mirror-token" {
		t.Fatalf("collector inputs spec=%#v token=%q", gotSpec, gotToken)
	}
	var decoded posture.MirrorInventory
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	if decoded.Kind != posture.MirrorInventoryKind || decoded.Version != posture.MirrorInventoryVersion || len(decoded.Repos) != 1 {
		t.Fatalf("mirror inventory envelope = %#v", decoded)
	}
}

func TestPostureMirrorsHelp(t *testing.T) {
	var stdout bytes.Buffer
	code := withStdout(t, &stdout, func() int {
		return run([]string{"posture", "mirrors", "--help"})
	})
	want := "usage: repoctl posture mirrors -f repora.yaml\n"
	if code != 0 || stdout.String() != want {
		t.Fatalf("help code=%d output=%q want=%q", code, stdout.String(), want)
	}
}
