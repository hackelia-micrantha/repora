package managedartifactapply

import (
	"testing"

	"repoctl/internal/journal"
)

func TestResultReportableOnlyAfterProjectedExecutionResult(t *testing.T) {
	tests := []struct {
		name string
		in   Result
		want bool
	}{
		{
			name: "early intent failure",
			in: Result{
				Version: ResultVersion, Kind: ResultKind, ExecutionID: "run-early",
				Journal: JournalReferences{Intent: ".repora/journal/managed-artifact--run-early--intent.json"},
			},
			want: false,
		},
		{
			name: "projected applied result",
			in: Result{
				Version: ResultVersion, Kind: ResultKind, ExecutionID: "run-applied", Outcome: journal.OutcomeApplied,
				Repositories: []RepositoryResult{{UID: "repo.demo", ID: "demo", Branch: "main", BaseOID: "1111111111111111111111111111111111111111", Pushed: true, Outcome: journal.OutcomeApplied}},
				Journal:      JournalReferences{Intent: ".repora/journal/managed-artifact--run-applied--intent.json"},
			},
			want: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.in.Reportable(); got != tc.want {
				t.Fatalf("Reportable() = %t, want %t", got, tc.want)
			}
		})
	}
}
