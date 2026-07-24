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

func TestReconcileIsDeterministicAndDoesNotMutateInputs(t *testing.T) {
	repo := testRepo()
	result := status.Result{State: status.StateDiverged, Ahead: 2, Behind: 4}
	observed := testObservation()
	repoBefore := cloneRepo(repo)
	resultBefore := result
	observedBefore := observed

	first, firstErr := Reconcile(repo, result, observed, true)
	second, secondErr := Reconcile(repo, result, observed, true)
	if firstErr != nil || secondErr != nil {
		t.Fatalf("Reconcile errors = %v, %v", firstErr, secondErr)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("determinism invariant violated:\nfirst: %#v\nsecond: %#v", first, second)
	}
	if !reflect.DeepEqual(repo, repoBefore) {
		t.Fatalf("topology input mutated:\nbefore: %#v\nafter: %#v", repoBefore, repo)
	}
	if !reflect.DeepEqual(result, resultBefore) {
		t.Fatalf("status input mutated:\nbefore: %#v\nafter: %#v", resultBefore, result)
	}
	if !reflect.DeepEqual(observed, observedBefore) {
		t.Fatalf("observation input mutated:\nbefore: %#v\nafter: %#v", observedBefore, observed)
	}
}

func TestReconcileActionOrderIsCanonicalToMirror(t *testing.T) {
	got, err := Reconcile(testRepo(), status.Result{State: status.StateBehind}, testObservation(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Actions) != 1 {
		t.Fatalf("actions = %#v, want exactly one ordered action", got.Actions)
	}
	action := got.Actions[0]
	if action.Source.Name != "canonical" || action.Source.Provider != "gitlab" {
		t.Fatalf("first action source = %#v, want configured canonical", action.Source)
	}
	if action.Target.Name != "mirror" || action.Target.Provider != "github" {
		t.Fatalf("first action target = %#v, want configured mirror", action.Target)
	}
}

func TestReconcileRejectsUnsupportedAndAmbiguousTopology(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*config.Repo)
		wantErr string
	}{
		{name: "missing id", mutate: func(repo *config.Repo) { repo.ID = " " }, wantErr: "requires a repo id"},
		{name: "unsupported canonical", mutate: func(repo *config.Repo) { repo.Canonical.Provider = "github" }, wantErr: "unsupported canonical provider"},
		{name: "no mirror", mutate: func(repo *config.Repo) { repo.Mirrors = nil }, wantErr: "exactly one mirror, got 0"},
		{name: "multiple mirrors", mutate: func(repo *config.Repo) { repo.Mirrors = append(repo.Mirrors, config.Endpoint{Provider: "gitlab"}) }, wantErr: "exactly one mirror, got 2"},
		{name: "unsupported mirror", mutate: func(repo *config.Repo) { repo.Mirrors[0].Provider = "bitbucket" }, wantErr: "unsupported mirror provider"},
		{name: "unsupported mode", mutate: func(repo *config.Repo) { repo.Mode = "sync" }, wantErr: "unsupported mode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testRepo()
			tt.mutate(&repo)
			first, firstErr := Reconcile(repo, status.Result{State: status.StateBehind}, testObservation(), false)
			second, secondErr := Reconcile(repo, status.Result{State: status.StateBehind}, testObservation(), false)
			if firstErr == nil || !strings.Contains(firstErr.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want actionable diagnostic containing %q; repo = %#v", firstErr, tt.wantErr, repo)
			}
			if secondErr == nil || secondErr.Error() != firstErr.Error() {
				t.Fatalf("diagnostic determinism violated: first = %v, second = %v; repo = %#v", firstErr, secondErr, repo)
			}
			if len(first.Actions) != 0 || len(second.Actions) != 0 {
				t.Fatalf("invalid topology produced partial executable plan: first = %#v, second = %#v", first.Actions, second.Actions)
			}
		})
	}
}

func TestReconcileSupportedTopologyCases(t *testing.T) {
	for _, mirrorProvider := range []string{"github", "gitlab"} {
		t.Run(mirrorProvider, func(t *testing.T) {
			repo := testRepo()
			repo.Mirrors[0].Provider = mirrorProvider
			got, err := Reconcile(repo, status.Result{State: status.StateBehind}, testObservation(), false)
			if err != nil {
				t.Fatalf("supported topology rejected: %v; repo = %#v", err, repo)
			}
			if len(got.Actions) != 1 || got.Actions[0].Target.Provider != mirrorProvider {
				t.Fatalf("plan = %#v, want action targeting %q", got, mirrorProvider)
			}
		})
	}
}

func TestReconcileConvergedObservationProducesNoActions(t *testing.T) {
	observed := testObservation()
	observed.MirrorHeadOID = observed.CanonicalHeadOID
	got, err := Reconcile(testRepo(), status.Result{State: status.StateEqual}, observed, false)
	if err != nil {
		t.Fatalf("converged plan returned error: %v", err)
	}
	if len(got.Actions) != 0 {
		t.Fatalf("convergence invariant violated: actions = %#v, want none", got.Actions)
	}
}

func TestReconcileRejectsInvalidInputsDeterministically(t *testing.T) {
	tests := []struct {
		name     string
		state    status.State
		observed Observation
		wantErr  string
	}{
		{name: "unsupported state", state: status.State("UNKNOWN"), observed: testObservation(), wantErr: "unsupported state"},
		{name: "missing branches", state: status.StateBehind, observed: Observation{CanonicalHeadOID: "source", MirrorHeadOID: "target"}, wantErr: "resolved canonical and mirror branches"},
		{name: "missing heads", state: status.StateBehind, observed: Observation{CanonicalBranch: "main", MirrorBranch: "main"}, wantErr: "observed canonical and mirror heads"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, firstErr := Reconcile(testRepo(), status.Result{State: tt.state}, tt.observed, false)
			second, secondErr := Reconcile(testRepo(), status.Result{State: tt.state}, tt.observed, false)
			if firstErr == nil || !strings.Contains(firstErr.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want containing %q; state = %q observed = %#v", firstErr, tt.wantErr, tt.state, tt.observed)
			}
			if secondErr == nil || secondErr.Error() != firstErr.Error() {
				t.Fatalf("diagnostic determinism violated: first = %v, second = %v", firstErr, secondErr)
			}
			if len(first.Actions) != 0 || len(second.Actions) != 0 {
				t.Fatalf("invalid input produced partial executable plan: first = %#v, second = %#v", first.Actions, second.Actions)
			}
		})
	}
}

func testRepo() config.Repo {
	return config.Repo{
		ID:        "payments-api",
		UID:       "repo.org.payments-api",
		Canonical: config.Endpoint{Provider: "gitlab"},
		Mirrors:   []config.Endpoint{{Provider: "github"}},
		Mode:      "mirror",
	}
}

func cloneRepo(repo config.Repo) config.Repo {
	clone := repo
	clone.Mirrors = append([]config.Endpoint(nil), repo.Mirrors...)
	return clone
}

func testObservation() Observation {
	return Observation{
		CanonicalBranch:  "main",
		CanonicalHeadOID: "source123456789",
		MirrorBranch:     "main",
		MirrorHeadOID:    "target123456789",
	}
}
