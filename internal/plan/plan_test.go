package plan

import (
	"reflect"
	"strings"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/status"
)

func TestNewRepoPlanAddsPushMirrorActionWhenMirrorIsBehind(t *testing.T) {
	got := NewRepoPlan(testRepo(), status.Result{State: status.StateBehind, Behind: 3})
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

func TestReconcileIsDeterministicAndDoesNotMutateInputs(t *testing.T) {
	repo := testRepo()
	result := status.Result{ID: repo.ID, State: status.StateDiverged, Ahead: 2, Behind: 3}
	observed := testObservation()
	repoBefore := cloneRepo(repo)
	resultBefore := result
	observedBefore := observed

	first, err := Reconcile(repo, result, observed, true)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Reconcile(repo, result, observed, true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("plans differ:\nfirst: %#v\nsecond: %#v", first, second)
	}
	if !reflect.DeepEqual(repo, repoBefore) || !reflect.DeepEqual(result, resultBefore) || !reflect.DeepEqual(observed, observedBefore) {
		t.Fatalf("planner mutated inputs: repo=%#v result=%#v observation=%#v", repo, result, observed)
	}
}

func TestReconcileProducesStableAction(t *testing.T) {
	got, err := Reconcile(testRepo(), status.Result{State: status.StateBehind}, testObservation(), false)
	if err != nil {
		t.Fatal(err)
	}
	want := []PlannedAction{{
		Type:              ActionPushBranch,
		Source:            Remote{Provider: "gitlab", Name: "canonical", Branch: "main"},
		Target:            Remote{Provider: "github", Name: "mirror", Branch: "main"},
		ExpectedSource:    "source123456789",
		ExpectedOldTarget: "target123456789",
		Reason:            "mirror is behind",
	}}
	if !reflect.DeepEqual(got.Actions, want) {
		t.Fatalf("actions = %#v, want stable action %#v", got.Actions, want)
	}
}

func TestReconcileRejectsUnsupportedTopologyWithoutPartialPlan(t *testing.T) {
	tests := []struct {
		name string
		edit func(*config.Repo)
		want string
	}{
		{name: "no mirror", edit: func(repo *config.Repo) { repo.Mirrors = nil }, want: "exactly one configured mirror"},
		{name: "multiple mirrors", edit: func(repo *config.Repo) { repo.Mirrors = append(repo.Mirrors, config.Endpoint{Provider: "gitlab"}) }, want: "exactly one configured mirror"},
		{name: "missing canonical provider", edit: func(repo *config.Repo) { repo.Canonical.Provider = "" }, want: "canonical and mirror providers"},
		{name: "missing mirror provider", edit: func(repo *config.Repo) { repo.Mirrors[0].Provider = "" }, want: "canonical and mirror providers"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepo()
			tt.edit(&repo)
			got, err := Reconcile(repo, status.Result{State: status.StateBehind}, testObservation(), false)
			if err == nil || !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), repo.ID) {
				t.Fatalf("error = %v, want repo identity and %q", err, tt.want)
			}
			if len(got.Actions) != 0 {
				t.Fatalf("actions = %#v, want no partial plan", got.Actions)
			}
		})
	}
}

func TestReconcileInvalidInputsFailDeterministically(t *testing.T) {
	tests := []struct {
		name     string
		state    status.State
		observed Observation
		want     string
	}{
		{name: "unsupported state", state: status.State("UNKNOWN"), observed: testObservation(), want: "unsupported state"},
		{name: "missing branch", state: status.StateBehind, observed: Observation{CanonicalHeadOID: "source", MirrorHeadOID: "target"}, want: "canonical and mirror branches"},
		{name: "missing source oid", state: status.StateBehind, observed: Observation{CanonicalBranch: "main", MirrorBranch: "main", MirrorHeadOID: "target"}, want: "canonical and mirror heads"},
		{name: "missing target oid", state: status.StateBehind, observed: Observation{CanonicalBranch: "main", CanonicalHeadOID: "source", MirrorBranch: "main"}, want: "canonical and mirror heads"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, firstErr := Reconcile(testRepo(), status.Result{State: tt.state}, tt.observed, false)
			second, secondErr := Reconcile(testRepo(), status.Result{State: tt.state}, tt.observed, false)
			if firstErr == nil || secondErr == nil || firstErr.Error() != secondErr.Error() || !strings.Contains(firstErr.Error(), tt.want) {
				t.Fatalf("errors = %v / %v, want deterministic error containing %q", firstErr, secondErr, tt.want)
			}
			if len(first.Actions) != 0 || len(second.Actions) != 0 {
				t.Fatalf("plans = %#v / %#v, want no actions", first, second)
			}
		})
	}
}

func TestReconcileConvergedObservationProducesNoActions(t *testing.T) {
	got, err := Reconcile(testRepo(), status.Result{State: status.StateEqual}, testObservation(), false)
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	if len(got.Actions) != 0 {
		t.Fatalf("actions = %#v, want converged no-op plan", got.Actions)
	}
}

func cloneRepo(repo config.Repo) config.Repo {
	clone := repo
	clone.Mirrors = append([]config.Endpoint(nil), repo.Mirrors...)
	return clone
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
