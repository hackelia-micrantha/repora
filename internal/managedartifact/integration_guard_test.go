package managedartifact

import "testing"

func requireIntegration(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test")
	}
}
