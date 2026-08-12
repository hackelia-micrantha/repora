package managedartifact

import (
	"strings"
	"testing"
)

func TestPlanRejectsMisleadingUnicodeTargetIdentity(t *testing.T) {
	for name, mutate := range map[string]func(*Plan){
		"provider path bidi": func(plan *Plan) { plan.Repositories[0].Target.Path = "micrantha/repo\u202e" },
		"branch format":      func(plan *Plan) { plan.Repositories[0].Target.Branch = "main\u2066" },
	} {
		t.Run(name, func(t *testing.T) {
			plan := validManagedPlan()
			mutate(&plan)
			if err := plan.Validate(); err == nil || !strings.Contains(err.Error(), "unsafe display character") {
				t.Fatalf("Validate() error = %v, want target display-safety rejection", err)
			}
		})
	}
}
