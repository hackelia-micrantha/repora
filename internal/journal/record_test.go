package journal

import (
	"bytes"
	"strings"
	"testing"

	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
)

const (
	testSourceOID = "1111111111111111111111111111111111111111"
	testTargetOID = "2222222222222222222222222222222222222222"
)

func TestRecordRoundTrip(t *testing.T) {
	record, err := FromPlan("run-001", ModeApply, testArtifact())
	if err != nil {
		t.Fatalf("FromPlan returned error: %v", err)
	}
	encoded, err := record.Marshal()
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	parsed, err := Parse(encoded)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if parsed.ExecutionID != "run-001" || parsed.Repository.UID != "repo.org.payments-api" || len(parsed.Actions) != 1 {
		t.Fatalf("parsed record = %#v", parsed)
	}
	if parsed.Actions[0].Before != testTargetOID || parsed.Actions[0].Desired != testSourceOID || parsed.Actions[0].Outcome != OutcomePlanned {
		t.Fatalf("action = %#v", parsed.Actions[0])
	}
}

func TestRecordJSONMatchesGoldenContract(t *testing.T) {
	record, err := FromPlan("run-001", ModeApply, testArtifact())
	if err != nil {
		t.Fatal(err)
	}
	got, err := record.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	want, err := testdata.ReadFile("testdata/execution-record-v2.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want)) {
		t.Fatalf("execution record contract changed:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestParseHistoricalVersion1Record(t *testing.T) {
	data, err := testdata.ReadFile("testdata/execution-record-v1.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	record, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if record.Version != LegacyVersion || record.Phase != "" || record.Actions[0].Outcome != OutcomePlanned {
		t.Fatalf("record = %#v", record)
	}
}

func TestParseRejectsUnknownAndTrailingFields(t *testing.T) {
	data, err := (Record{
		Version:     LegacyVersion,
		Kind:        Kind,
		ExecutionID: "run",
		Mode:        ModeApply,
		Plan:        PlanRef{Version: planartifact.LegacyVersion, Kind: planartifact.Kind, SHA256: strings.Repeat("a", 64)},
		Repository:  Repository{UID: "repo.uid", ID: "repo"},
		Actions:     []Action{},
	}).Marshal()
	if err != nil {
		t.Fatal(err)
	}
	withUnknown := append(data[:len(data)-1], []byte(`,"url":"https://example.com"}`)...)
	if _, err := Parse(withUnknown); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field error = %v", err)
	}
	if _, err := Parse(append(data, []byte(` {}`)...)); err == nil || !strings.Contains(err.Error(), "trailing") {
		t.Fatalf("trailing error = %v", err)
	}
}

func TestRecordRejectsMultipleRepositories(t *testing.T) {
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
		{name: "version", edit: func(r *Record) { r.Version = 99 }, want: "version"},
		{name: "kind", edit: func(r *Record) { r.Kind = "unknown" }, want: "kind"},
		{name: "execution id", edit: func(r *Record) { r.ExecutionID = "/tmp/run" }, want: "execution_id"},
		{name: "phase", edit: func(r *Record) { r.Phase = "UNKNOWN" }, want: "phase"},
		{name: "mode", edit: func(r *Record) { r.Mode = "UNKNOWN" }, want: "mode"},
		{name: "digest", edit: func(r *Record) { r.Plan.SHA256 = "not-a-digest" }, want: "plan reference"},
		{name: "identity", edit: func(r *Record) { r.Repository.UID = "https://example.com/repo" }, want: "uid and id"},
		{name: "index", edit: func(r *Record) { r.Actions[0].Index = 7 }, want: "index"},
		{name: "before oid", edit: func(r *Record) { r.Actions[0].Before = "not-an-oid" }, want: "object IDs"},
		{name: "after oid", edit: func(r *Record) { r.Actions[0].After = "not-an-oid" }, want: "after object ID"},
		{name: "outcome", edit: func(r *Record) { r.Actions[0].Outcome = "UNKNOWN" }, want: "outcome"},
		{name: "intent result", edit: func(r *Record) { r.Actions[0].Outcome = OutcomeSkipped }, want: "intent entry"},
		{name: "secret error", edit: func(r *Record) {
			r.Phase = PhaseResult
			r.Actions[0].Outcome = OutcomeFailed
			r.Actions[0].Error = "token=secret"
		}, want: "unsafe"},
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

func TestResultPhaseRules(t *testing.T) {
	record, err := FromPlan("run-001", ModeApply, testArtifact())
	if err != nil {
		t.Fatal(err)
	}
	record.Phase = PhaseResult
	record.Actions[0].Outcome = OutcomeApplied
	record.Actions[0].After = testSourceOID
	encoded, err := record.Marshal()
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	if !strings.Contains(string(encoded), `"phase": "RESULT"`) || !strings.Contains(string(encoded), `"after": "`+testSourceOID+`"`) {
		t.Fatalf("record missing result evidence:\n%s", encoded)
	}

	dryRun := record
	dryRun.Mode = ModeDryRun
	if err := dryRun.Validate(); err == nil || !strings.Contains(err.Error(), "dry-run") {
		t.Fatalf("dry-run applied error = %v", err)
	}

	applyValidated := record
	applyValidated.Actions[0].Outcome = OutcomeValidated
	applyValidated.Actions[0].After = ""
	if err := applyValidated.Validate(); err == nil || !strings.Contains(err.Error(), "apply") {
		t.Fatalf("apply validated error = %v", err)
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
			Reason:            "mirror is behind",
		}},
	}
}
