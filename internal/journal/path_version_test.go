package journal

import (
	"strings"
	"testing"

	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
)

func TestPathVersionRequiresPlanArtifactV2(t *testing.T) {
	artifact, err := planartifact.FromCurrentPlans(plan.ReconciliationPlan{
		ID:  "payments-api",
		UID: "repo.org.payments-api",
		Actions: []plan.PlannedAction{{
			Type: plan.ActionPushBranch,
			Source: plan.Remote{
				Provider: "gitlab",
				Path:     "org/payments-api",
				Name:     "canonical",
				Branch:   "main",
			},
			Target: plan.Remote{
				Provider: "github",
				Path:     "org/payments-api",
				Name:     "mirror-0",
				Branch:   "main",
			},
			ExpectedSource:    testSourceOID,
			ExpectedOldTarget: testTargetOID,
			Reason:            "mirror is behind",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	record, err := FromPlan("run-path", ModeDryRun, artifact)
	if err != nil {
		t.Fatal(err)
	}
	if record.Version != PathVersion || record.Plan.Version != planartifact.Version {
		t.Fatalf("record = %#v, want execution v3 referencing plan v2", record)
	}

	record.Plan.Version = planartifact.LegacyVersion
	if err := record.Validate(); err == nil || !strings.Contains(err.Error(), "requires plan artifact version") {
		t.Fatalf("error = %v, want v3/v2 binding rejection", err)
	}
}
