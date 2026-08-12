package managedartifact

import (
	"strings"
	"testing"
)

func TestRenderREADMERejectsUnicodeDisplayControls(t *testing.T) {
	for _, value := range []string{"safe\u202eunsafe", "safe\u2028unsafe"} {
		_, err := RenderREADME([]byte("{{value.text}}"), RenderData{Values: map[string]string{"text": value}})
		if err == nil || !strings.Contains(err.Error(), "unsafe display character") {
			t.Fatalf("RenderREADME(%q) error = %v, want display-control rejection", value, err)
		}
	}
}

func TestPlanRejectsUnicodeDisplayControls(t *testing.T) {
	t.Run("desired content", func(t *testing.T) {
		plan := validManagedPlan()
		content := "safe\u202eunsafe\n"
		plan.Repositories[0].Actions[0].Desired.Content = stringPointer(content)
		plan.Repositories[0].Actions[0].Desired.SHA256 = DigestSHA256([]byte(content))
		if err := plan.Validate(); err == nil || !strings.Contains(err.Error(), "unsafe display character") {
			t.Fatalf("Validate() error = %v, want display-control rejection", err)
		}
	})

	t.Run("review diff", func(t *testing.T) {
		plan := validManagedPlan()
		plan.Repositories[0].Actions[0].Diff = "--- a/README.md\n+++ b/README.md\n@@ -1 +1 @@\n-old\n+safe\u2028unsafe\n"
		if err := plan.Validate(); err == nil || !strings.Contains(err.Error(), "unsafe display character") {
			t.Fatalf("Validate() error = %v, want display-control rejection", err)
		}
	})
}
