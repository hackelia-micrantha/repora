package journal

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
)

const (
	testSourceOID = "1111111111111111111111111111111111111111"
	testTargetOID = "2222222222222222222222222222222222222222"
)

func TestFromPlanCreatesDeterministicReferenceAndActions(t *testing.T) {
	artifact := testArtifact()
	first, err := FromPlan("run-001", ModeDryRun, artifact)
	if err != nil {
		t.Fatalf("FromPlan returned error: %v", err)
	}
	second, err := FromPlan("run-001", ModeDryRun, artifact)
	if err != nil {
		t.Fatalf("FromPlan returned error: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("records differ:\n%#v\n%#v", first, second)
	}
	if first.Plan.SHA256 == "" || len(first.Plan.SHA256) != 64 {
		t.Fatalf("plan digest = %q, want SHA-256", first.Plan.SHA256)
	}
	if len(first.Actions) != 1 {
		t.Fatalf("actions = %#v, want one", first.Actions)
	}
	action := first.Actions[0]
	if action.Index != 0 || action.Before != testTargetOID || action.Desired != testSourceOID || action.Outcome != OutcomePlanned {
		t.Fatalf("action = %#v, want ordered planned ref evidence", action)
	}
}

func TestRecordSerializationIsDeterministicAndRoundTrips(t *testing.T) {
	record, err := FromPlan("run-001", ModePlan, testArtifact())
	if err != nil {
		t.Fatal(err)
	}
	first, err := record.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	second, err := record.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("serialization differs:\n%s\n%s", first, second)
	}
	decoded, err := Parse(first)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !reflect.DeepEqual(decoded, record) {
		t.Fatalf("decoded = %#v, want %#v", decoded, record)
	}
}

func TestFromPlanRejectsInvalidOrMultiRepositoryArtifact(t *testing.T) {
	invalid := testArtifact()
	invalid.Version = 99
	if _, err := FromPlan("run-001", ModeApply, invalid); err == nil || !strings.Contains(err.Error(), "serialize plan artifact") {
		t.Fatalf("error = %v, want artifact validation failure", err)
	}

	multi := planartifact.FromPlans(testPlan(), testPlan())
	if _, err := FromPlan("run-001", ModeApply, multi); err == nil || !strings.Contains(err.Error(), "exactly one repository") {
		t.Fatalf("error = %v, want repository cardinality failure", err)
	}
}

func TestRecordRejectsInvalidEnvelopeAndSafetyFields(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Record)
		want string
	}{
		{name: "version", edit: func(r *Record) { r.Version = 2 }, want: "version"},
		{name: "kind", edit: func(r *Record) { r.Kind = "unknown" }, want: "kind"},
		{name: "execution id", edit: func(r *Record) { r.ExecutionID = "/tmp/run" }, want: "execution_id"},
		{name: "mode", edit: func(r *Record) { r.Mode = "UNKNOWN" }, want: "mode"},
		{name: "digest", edit: func(r *Record) { r.Plan.SHA256 = "not-a-digest" }, want: "plan reference"},
		{name: "identity", edit: func(r *Record) { r.Repository.UID = "https://example.com/repo" }, want: "uid and id"},
		{name: "index", edit: func(r *Record) { r.Actions[0].Index = 7 }, want: "index"},
		{name: "before oid", edit: func(r *Record) { r.Actions[0].Before = "not-an-oid" }, want: "object IDs"},
		{name: "after oid", edit: func(r *Record) { r.Actions[0].After = "not-an-oid" }, want: "after object ID"},
		{name: "outcome", edit: func(r *Record) { r.Actions[0].Outcome = "UNKNOWN" }, want: "outcome"},
		{name: "applied without after", edit: func(r *Record) { r.Actions[0].Outcome = OutcomeApplied }, want: "requires after"},
		{name: "secret error", edit: func(r *Record) { r.Actions[0].Outcome = OutcomeFailed; r.Actions[0].Error = "token=secret" }, want: "unsafe"},
		{name: "transport ref", edit: func(r *Record) { r.Actions[0].Target.Remote = "git@github.com:org/repo" }, want: "symbolic"},
		{name: "branch with spaces", edit: func(r *Record) { r.Actions[0].Target.Branch = "release candidate" }, want: "symbolic ref"},
		{name: "branch traversal", edit: func(r *Record) { r.Actions[0].Target.Branch = "release..next" }, want: "symbolic ref"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := FromPlan("run-001", ModeApply, testArtifact())
			if err != nil {
				t.Fatal(err)
			}
			tt.edit(&record)
			if err := record.Validate(); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestAppliedActionRequiresAndPreservesAfterOID(t *testing.T) {
	record, err := FromPlan("run-001", ModeApply, testArtifact())
	if err != nil {
		t.Fatal(err)
	}
	record.Actions[0].Outcome = OutcomeApplied
	record.Actions[0].After = testSourceOID
	encoded, err := record.Marshal()
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	if !strings.Contains(string(encoded), `"after": "`+testSourceOID+`"`) {
		t.Fatalf("record missing after OID:\n%s", encoded)
	}
}

func TestParseRejectsUnknownFieldsAndTrailingData(t *testing.T) {
	record, err := FromPlan("run-001", ModePlan, testArtifact())
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := record.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	unknown := bytes.Replace(encoded, []byte(`"mode": "PLAN"`), []byte(`"mode": "PLAN", "secret": "x"`), 1)
	if _, err := Parse(unknown); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("error = %v, want unknown-field rejection", err)
	}
	for _, suffix := range []string{`{"extra":true}`, ` trailing`} {
		payload := append(append([]byte(nil), encoded...), []byte(suffix)...)
		if _, err := Parse(payload); err == nil || !strings.Contains(err.Error(), "trailing") {
			t.Fatalf("suffix %q error = %v, want trailing-data rejection", suffix, err)
		}
	}
}

func testArtifact() planartifact.Artifact {
	return planartifact.FromPlans(testPlan())
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
