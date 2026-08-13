package managedartifact

import (
	"strings"
	"testing"
)

func TestPlanPreservesLegacyNegativeContractCoverage(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Plan)
		wantErr string
	}{
		{
			name: "unsupported kind",
			mutate: func(plan *Plan) {
				plan.Kind = "other"
			},
			wantErr: "unsupported managed artifact plan contract",
		},
		{
			name: "invalid base oid",
			mutate: func(plan *Plan) {
				plan.Repositories[0].BaseOID = "deadbeef"
			},
			wantErr: "base_oid",
		},
		{
			name: "multiple actions",
			mutate: func(plan *Plan) {
				action := plan.Repositories[0].Actions[0]
				plan.Repositories[0].Actions = append(plan.Repositories[0].Actions, action)
			},
			wantErr: "exactly one README action",
		},
		{
			name: "unsupported action type",
			mutate: func(plan *Plan) {
				plan.Repositories[0].Actions[0].Type = "WRITE_FILE"
			},
			wantErr: "unsupported action type",
		},
		{
			name: "wrong output path",
			mutate: func(plan *Plan) {
				plan.Repositories[0].Actions[0].Path = "docs/README.md"
			},
			wantErr: "managed artifact path",
		},
		{
			name: "missing observed present",
			mutate: func(plan *Plan) {
				plan.Repositories[0].Actions[0].Observed.Present = nil
			},
			wantErr: "observed present field is required",
		},
		{
			name: "present observed state missing digest",
			mutate: func(plan *Plan) {
				plan.Repositories[0].Actions[0].Observed.SHA256 = ""
			},
			wantErr: "observed README sha256",
		},
		{
			name: "uppercase desired digest",
			mutate: func(plan *Plan) {
				action := &plan.Repositories[0].Actions[0]
				action.Desired.SHA256 = strings.ToUpper(action.Desired.SHA256)
			},
			wantErr: "lowercase SHA-256",
		},
		{
			name: "invalid template digest",
			mutate: func(plan *Plan) {
				plan.Repositories[0].Actions[0].TemplateSHA256 = "abc"
			},
			wantErr: "template_sha256",
		},
		{
			name: "CR desired content",
			mutate: func(plan *Plan) {
				action := &plan.Repositories[0].Actions[0]
				content := "# Repora\r\n"
				action.Desired.Content = &content
				action.Desired.SHA256 = DigestSHA256([]byte(content))
			},
			wantErr: "LF line endings",
		},
		{
			name: "oversized desired content",
			mutate: func(plan *Plan) {
				action := &plan.Repositories[0].Actions[0]
				content := strings.Repeat("x", MaxTextBytes+1)
				action.Desired.Content = &content
				action.Desired.SHA256 = DigestSHA256([]byte(content))
			},
			wantErr: "content exceeds",
		},
		{
			name: "oversized diff",
			mutate: func(plan *Plan) {
				plan.Repositories[0].Actions[0].Diff = "--- a/README.md\n+++ b/README.md\n@@ -1 +1 @@\n" + strings.Repeat("x", MaxDiffBytes)
			},
			wantErr: "review diff exceeds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := validManagedPlan()
			tt.mutate(&plan)
			err := plan.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}
