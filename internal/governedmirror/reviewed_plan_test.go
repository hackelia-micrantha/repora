package governedmirror

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"repoctl/internal/planartifact"
)

func samplePlan(t *testing.T) []byte {
	t.Helper()
	a := planartifact.Artifact{
		Kind:    planartifact.Kind,
		Version: planartifact.Version,
		Repositories: []planartifact.Repository{{
			UID: "repo.sample",
			ID:  "repo.sample",
			Actions: []planartifact.Action{{
				Type: "PUSH_BRANCH",
				Source: planartifact.Ref{
					Provider: "gitlab", Path: "team/repo", Remote: "canonical", Branch: "main",
				},
				Target: planartifact.Ref{
					Provider: "github", Path: "team/repo", Remote: "mirror", Branch: "main",
				},
				Diff: planartifact.RefDiff{
					Observed: strings.Repeat("a", 40), Desired: strings.Repeat("b", 40),
				},
				Force: false, Reason: "behind",
			}},
		}},
	}
	raw, err := a.Marshal()
	if err != nil {
		t.Fatalf("marshal sample: %v", err)
	}
	return raw
}

func TestReviewedPlanExactBytes(t *testing.T) {
	raw := samplePlan(t)
	got, err := ParseReviewedPlan(raw, false)
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(raw)
	if got.RequiresForce {
		t.Fatal("non-destructive plan was classified as requiring force")
	}
	if got.Digest != hex.EncodeToString(want[:]) {
		t.Fatalf("digest %q does not bind exact plan bytes", got.Digest)
	}
	// Harmless formatting still creates a different exact reviewed artifact.
	reformatted := append(append([]byte{}, raw...), '\n')
	changed, err := ParseReviewedPlan(reformatted, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Digest == changed.Digest {
		t.Fatal("reformatted bytes unexpectedly retained the same digest")
	}
}

func TestReviewedPlanRejectsAmbiguity(t *testing.T) {
	raw := string(samplePlan(t))
	cases := map[string][]byte{
		"duplicate-top-level": []byte(strings.Replace(raw, "\"version\": 2", "\"version\": 2, \"version\": 2", 1)),
		"duplicate-nested":    []byte(strings.Replace(raw, "\"force\": false", "\"force\": false, \"force\": true", 1)),
		"escaped-duplicate":   []byte(strings.Replace(raw, "\"force\": false", "\"force\": false, \"fo\\u0072ce\": true", 1)),
		"multiple-values":     []byte(raw + " {}"),
		"unknown-actor":       []byte(strings.Replace(raw, "\"version\": 2", "\"version\": 2, \"actor\": \"admin\"", 1)),
		"legacy-version":      []byte(strings.Replace(raw, "\"version\": 2", "\"version\": 1", 1)),
		"malformed":           []byte("{"),
		"invalid-utf8":        {0xff, 0xfe},
		"empty":               {},
		"too-large":           []byte(strings.Repeat(" ", MaxArtifactBytes+1)),
	}
	for name, candidate := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseReviewedPlan(candidate, false); err == nil {
				t.Fatal("ambiguous or unsupported reviewed plan was accepted")
			}
		})
	}
}

func TestReviewedPlanScopeAndForceIntent(t *testing.T) {
	base := samplePlan(t)
	artifact, err := planartifact.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	artifact.Repositories[0].Actions[0].Force = true
	forced, err := artifact.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseReviewedPlan(forced, false); err == nil {
		t.Fatal("forced action accepted without explicit destructive intent")
	}
	reviewedForce, err := ParseReviewedPlan(forced, true)
	if err != nil {
		t.Fatalf("reviewed force plus explicit intent should permit read-only preparation: %v", err)
	}
	if !reviewedForce.RequiresForce {
		t.Fatal("force classification lost after review")
	}
	artifact.Repositories[0].Actions = append(artifact.Repositories[0].Actions, artifact.Repositories[0].Actions[0])
	twoActions, err := artifact.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseReviewedPlan(twoActions, true); err == nil {
		t.Fatal("multi-action plan accepted by single-mirror pilot")
	}
	artifact.Repositories[0].Actions = artifact.Repositories[0].Actions[:1]
	artifact.Repositories = append(artifact.Repositories, artifact.Repositories[0])
	twoRepos, err := artifact.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseReviewedPlan(twoRepos, true); err == nil {
		t.Fatal("multi-repository plan accepted by pilot")
	}
	artifact.Repositories = nil
	emptyPlan, err := artifact.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseReviewedPlan(emptyPlan, true); err == nil {
		t.Fatal("empty plan accepted for governed effect")
	}
}

func TestReviewedPlanRejectsDeeplyNestedJSON(t *testing.T) {
	nested := strings.Repeat("[", maxJSONDepth+2) + "null" + strings.Repeat("]", maxJSONDepth+2)
	if _, err := ParseReviewedPlan([]byte(nested), false); err == nil {
		t.Fatal("unexpectedly accepted unbounded nesting")
	}
}

func FuzzReviewedPlanDoesNotPanic(f *testing.F) {
	// Fuzz only the bounded, credential-free decoder; the result never grants authority.
	f.Add([]byte("{}"), false)
	f.Add([]byte("{\"version\":2,\"version\":2}"), false)
	f.Add([]byte("{\"force\":true}"), true)
	f.Fuzz(func(t *testing.T, raw []byte, intent bool) {
		if len(raw) > MaxArtifactBytes {
			return
		}
		_, _ = ParseReviewedPlan(raw, intent)
	})
}
