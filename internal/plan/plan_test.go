package plan

import (
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/status"
)

func TestNewRepoPlanAddsPushMirrorActionWhenMirrorIsBehind(t *testing.T) {
	repo := config.Repo{
		ID: "payments-api",
		Mirrors: []config.Endpoint{
			{Provider: "github", URL: "git@github.com:org/payments-api.git"},
		},
	}
	result := status.Result{
		ID:     "payments-api",
		State:  status.StateBehind,
		Behind: 3,
	}

	got := NewRepoPlan(repo, result)

	if got.ID != "payments-api" {
		t.Fatalf("id = %q, want payments-api", got.ID)
	}
	if len(got.Actions) != 1 {
		t.Fatalf("action count = %d, want 1", len(got.Actions))
	}
	action := got.Actions[0]
	if action.Type != "PUSH_MIRROR" || action.Target != "github" || action.Behind != 3 || action.Destructive {
		t.Fatalf("action = %#v, want non-destructive PUSH_MIRROR to github behind 3", action)
	}
}

func TestNewRepoPlanAddsNoActionsWhenMirrorIsEqual(t *testing.T) {
	repo := config.Repo{
		ID: "payments-api",
		Mirrors: []config.Endpoint{
			{Provider: "github", URL: "git@github.com:org/payments-api.git"},
		},
	}
	result := status.Result{ID: "payments-api", State: status.StateEqual}

	got := NewRepoPlan(repo, result)

	if len(got.Actions) != 0 {
		t.Fatalf("actions = %#v, want none", got.Actions)
	}
}
