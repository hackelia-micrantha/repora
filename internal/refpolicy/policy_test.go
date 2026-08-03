package refpolicy

import (
	"strings"
	"testing"
)

func TestNormalizeDefaultsClosedPolicy(t *testing.T) {
	got, err := (Policy{}).Normalize()
	if err != nil {
		t.Fatalf("Normalize returned error: %v", err)
	}
	want := Default()
	if got != want {
		t.Fatalf("policy = %#v, want %#v", got, want)
	}
}

func TestNormalizeRejectsUnsupportedValues(t *testing.T) {
	tests := []struct {
		name   string
		policy Policy
		want   string
	}{
		{name: "version", policy: Policy{Version: 2}, want: "version"},
		{name: "scope", policy: Policy{Scope: "all-branches"}, want: "scope"},
		{name: "destructive", policy: Policy{Destructive: "allow"}, want: "destructive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.policy.Normalize(); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestDecideRelationship(t *testing.T) {
	tests := []struct {
		relationship Relationship
		action       bool
		force        bool
	}{
		{relationship: RelationshipEqual},
		{relationship: RelationshipBehind, action: true},
		{relationship: RelationshipAhead, action: true, force: true},
		{relationship: RelationshipDiverged, action: true, force: true},
	}
	for _, tt := range tests {
		decision, err := Default().Decide(tt.relationship)
		if err != nil {
			t.Fatalf("Decide(%s) returned error: %v", tt.relationship, err)
		}
		if decision.Action != tt.action || decision.Force != tt.force {
			t.Fatalf("Decide(%s) = %#v, want action=%v force=%v", tt.relationship, decision, tt.action, tt.force)
		}
	}
}

func TestValidateDefaultBranches(t *testing.T) {
	if err := Default().ValidateDefaultBranches("main", "main", "main", "main"); err != nil {
		t.Fatalf("ValidateDefaultBranches returned error: %v", err)
	}
	if err := Default().ValidateDefaultBranches("release", "main", "main", "main"); err == nil || !strings.Contains(err.Error(), "current default branches") {
		t.Fatalf("error = %v, want scope mismatch", err)
	}
}
