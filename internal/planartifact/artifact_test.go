package planartifact

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"repoctl/internal/plan"
)

const (
	testSourceOID = "1111111111111111111111111111111111111111"
	testTargetOID = "2222222222222222222222222222222222222222"
)

func TestArtifactRoundTripPreservesPlan(t *testing.T) {
	original := testPlan()
	artifact := FromPlans(original)

	encoded, err := artifact.Marshal()
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	decoded, err := Parse(encoded)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	plans, err := decoded.Plans()
	if err != nil {
		t.Fatalf("Plans returned error: %v", err)
	}
	if len(plans) != 1 || !reflect.DeepEqual(plans[0], original) {
		t.Fatalf("round trip = %#v, want %#v", plans, original)
	}
}

func TestArtifactMatchesGoldenContract(t *testing.T) {
	got, err := FromPlans(testPlan()).Marshal()
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "reconciliation-plan-v1.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(append(got, '\n'), want) {
		t.Fatalf("plan artifact contract changed:\n%s\nwant:\n%s", got, want)
	}
}

func TestArtifactSerializationIsDeterministic(t *testing.T) {
	artifact := FromPlans(testPlan())
	first, err := artifact.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	second, err := artifact.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("serialization differs:\n%s\n%s", first, second)
	}
}

func TestArtifactContainsVersionKindIdentityAndRefDiff(t *testing.T) {
	encoded, err := FromPlans(testPlan()).Marshal()
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, want := range []string{
		`"version": 1`,
		`"kind": "repora.io/reconciliation-plan"`,
		`"uid": "repo.org.payments-api"`,
		`"observed": "` + testTargetOID + `"`,
		`"desired": "` + testSourceOID + `"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("artifact missing %s:\n%s", want, text)
		}
	}
}

func TestArtifactRejectsInvalidEnvelopeAndAction(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Artifact)
		want string
	}{
		{name: "version", edit: func(a *Artifact) { a.Version = 2 }, want: "version"},
		{name: "kind", edit: func(a *Artifact) { a.Kind = "unknown" }, want: "kind"},
		{name: "identity", edit: func(a *Artifact) { a.Repositories[0].UID = "" }, want: "uid and id"},
		{name: "action type", edit: func(a *Artifact) { a.Repositories[0].Actions[0].Type = "DELETE" }, want: "unsupported type"},
		{name: "source oid", edit: func(a *Artifact) { a.Repositories[0].Actions[0].Diff.Desired = "not-an-oid" }, want: "object IDs"},
		{name: "target oid", edit: func(a *Artifact) { a.Repositories[0].Actions[0].Diff.Observed = "https://example.com/repo" }, want: "object IDs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifact := FromPlans(testPlan())
			tt.edit(&artifact)
			if err := artifact.Validate(); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestArtifactRejectsUnknownJSONFields(t *testing.T) {
	_, err := Parse([]byte(`{"version":1,"kind":"repora.io/reconciliation-plan","repositories":[],"secret":"x"}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("error = %v, want unknown-field rejection", err)
	}
}

func TestArtifactRejectsTrailingData(t *testing.T) {
	valid := `{"version":1,"kind":"repora.io/reconciliation-plan","repositories":[]}`
	for _, suffix := range []string{
		`{"extra":true}`,
		` trailing`,
	} {
		_, err := Parse([]byte(valid + suffix))
		if err == nil || !strings.Contains(err.Error(), "trailing") {
			t.Fatalf("suffix %q error = %v, want trailing-data rejection", suffix, err)
		}
	}
}

func TestArtifactRejectsSensitiveOrRuntimeLocationData(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Artifact)
	}{
		{name: "tokenized URL", edit: func(a *Artifact) { a.Repositories[0].Actions[0].Source.Provider = "https://token@example.com" }},
		{name: "scp URL", edit: func(a *Artifact) { a.Repositories[0].Actions[0].Source.Remote = "git@github.com:org/repo.git" }},
		{name: "query token", edit: func(a *Artifact) { a.Repositories[0].Actions[0].Reason = "token=secret" }},
		{name: "absolute path", edit: func(a *Artifact) { a.Repositories[0].Actions[0].Target.Remote = "/tmp/mirror" }},
		{name: "file URL", edit: func(a *Artifact) { a.Repositories[0].ID = "file:///tmp/repo" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifact := FromPlans(testPlan())
			tt.edit(&artifact)
			if _, err := artifact.Marshal(); err == nil {
				t.Fatal("Marshal returned nil error for sensitive data")
			}
		})
	}
}

func TestArtifactAcceptsConservativeSymbolicNames(t *testing.T) {
	artifact := FromPlans(testPlan())
	action := &artifact.Repositories[0].Actions[0]
	action.Source.Provider = "gitlab-self_hosted"
	action.Source.Remote = "canonical.prod"
	action.Source.Branch = "release/2026-07"
	if err := artifact.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func testPlan() plan.ReconciliationPlan {
	return plan.ReconciliationPlan{
		ID:  "payments-api",
		UID: "repo.org.payments-api",
		Actions: []plan.PlannedAction{{
			Type:              plan.ActionPushBranch,
			Source:            plan.Remote{Provider: "gitlab", Name: "canonical", Branch: "main"},
			Target:            plan.Remote{Provider: "github", Name: "mirror", Branch: "main"},
			ExpectedSource:    testSourceOID,
			ExpectedOldTarget: testTargetOID,
			Force:             true,
			Reason:            "mirror is diverged",
		}},
	}
}
