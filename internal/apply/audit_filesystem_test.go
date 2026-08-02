package apply

import (
	"os"
	"path/filepath"
	"testing"

	"repoctl/internal/journal"
	"repoctl/internal/status"
)

func TestExecuteArtifactAuditedPersistsParseablePhaseFiles(t *testing.T) {
	root := t.TempDir()
	repo := testRepo()
	got, err := ExecuteArtifactAudited(repo, status.Result{State: status.StateBehind}, auditedArtifact(repo, false), &fakeGit{}, false, true, Audit{
		ExecutionID: "run-filesystem",
		Writer:      journal.Writer{Root: root},
	})
	if err != nil {
		t.Fatalf("ExecuteArtifactAudited returned error: %v", err)
	}
	if got.Journal == nil || got.Journal.Intent == "" || got.Journal.Result == "" {
		t.Fatalf("journal references = %#v, want intent and result", got.Journal)
	}

	for reference, phase := range map[string]journal.Phase{
		got.Journal.Intent: journal.PhaseIntent,
		got.Journal.Result: journal.PhaseResult,
	} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(reference)))
		if err != nil {
			t.Fatalf("read %s: %v", reference, err)
		}
		record, err := journal.Parse(data)
		if err != nil {
			t.Fatalf("parse %s: %v", reference, err)
		}
		if record.ExecutionID != "run-filesystem" || record.Phase != phase {
			t.Fatalf("record = %#v, want execution run-filesystem phase %s", record, phase)
		}
	}
}
