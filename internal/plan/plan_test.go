package plan

import (
	"reflect"
	"strings"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/status"
)

func TestNewRepoPlanAddsPushMirrorActionWhenMirrorIsBehind(t *testing.T) {
	repo := testRepo()
	result := status.Result{State: status.StateBehind, Behind: 3}
	got := NewRepoPlan(repo, result)
	if len(got.Actions) != 1 || got.Actions[0].Type != "PUSH_MIRROR" {
		t.Fatalf("actions = %#v, want legacy PUSH_MIRROR action", got.Actions)
	}
}

func TestReconcileStates(t *testing.T) {
	tests := []struct {
		name       string
		state      status.State
		force      bool
		wantAction bool
		wantForce  bool
		wantErr    string
	}{
		{name: "equal", state: status.StateEqual},
		{name: "behind", state: status.StateBehind, wantAction: true},
		{name: "ahead fails closed", state: status.StateAhead, wantAction: true, wantForce: true, wantErr: "--force"},
		{name: "diverged fails closed", state: status.StateDiverged, wantAction: true, wantForce: true, wantErr: "--force"},
		{name: "forced ahead", state: status.StateAhead, force: true, wantAction: true, wantForce: true},
		{name: "forced diverged", state: status.StateDiverged, force: true, wantAction: true, wantForce: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Reconcile(testRepo(), status.Result{State: tt.state}, testObservation(), tt.force)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("Reconcile returned error: %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
			if (len(got.Actions) == 1) != tt.wantAction {
				t.Fatalf("actions = %#v, wantAction = %v", got.Actions, tt.wantAction)
			}
			if tt.wantAction {
				action := got.Actions[0]
				if action.Force != tt.wantForce {
					t.Fatalf("force = %v, want %v", action.Force, tt.wantForce)
				}
				if action.ExpectedSource != "source123456789" || action.ExpectedOldTarget != "target123456789" {
					t.Fatalf("action = %#v, want captured source and target refs", action)
				}
			}
		})
	}
}

func TestReconcileRequiresObservedRefsForMutation(t *testing.T) {
	observed := testObservation()
	observed.CanonicalHeadOID = ""
	got, err := Reconcile(testRepo(), status.Result{State: status.StateBehind}, observed, false)
	if err == nil || !strings.Contains(err.Error(), "canonical and mirror heads") {
		t.Fatalf("error = %v, want missing observed refs", err)
	}
	if len(got.Actions) != 0 {
		t.Fatalf("actions = %#v, want none", got.Actions)
	}
}

func TestReconcileIsDeterministic(t *testing.T) {
	first, err := Reconcile(testRepo(), status.Result{State: status.StateDiverged}, testObservation(), true)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Reconcile(testRepo(), status.Result{State: status.StateDiverged}, testObservation(), true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("plans differ:\nfirst: %#v\nsecond: %#v", first, second)
	}
}

func TestReconcileRejectsUnsupportedState(t *testing.T) {
	got, err := Reconcile(testRepo(), status.Result{State: status.State("UNKNOWN")}, testObservation(), false)
	if err == nil {
		t.Fatal("Reconcile returned nil error")
	}
	if len(got.Actions) != 0 {
		t.Fatalf("actions = %#v, want none", got.Actions)
	}
}

func testRepo() config.Repo {
	return config.Repo{
		ID:        "payments-api",
		UID:       "repo.org.payments-api",
		Canonical: config.Endpoint{Provider: "gitlab"},
		Mirrors:   []config.Endpoint{{Provider: "github"}},
	}
}

func testObservation() Observation {
	return Observation{
		CanonicalBranch:  "main",
		CanonicalHeadOID: "source123456789",
		MirrorBranch:     "main",
		MirrorHeadOID:    "target123456789",
	}
}
