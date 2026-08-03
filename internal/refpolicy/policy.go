// Package refpolicy defines the closed synchronization policy supported by the
// current mirror-controller runtime.
package refpolicy

import (
	"fmt"
	"strings"
)

const Version = 1

type Scope string

type DestructiveMode string

type Relationship string

const (
	ScopeDefaultBranchOnly Scope = "default-branch-only"

	DestructiveRequireForce DestructiveMode = "require-force"

	RelationshipEqual    Relationship = "EQUAL"
	RelationshipBehind   Relationship = "BEHIND"
	RelationshipAhead    Relationship = "AHEAD"
	RelationshipDiverged Relationship = "DIVERGED"
)

type Policy struct {
	Version     int             `json:"version" yaml:"version"`
	Scope       Scope           `json:"scope" yaml:"scope"`
	Destructive DestructiveMode `json:"destructive" yaml:"destructive"`
}

type Decision struct {
	Action bool
	Force  bool
	Reason string
}

func Default() Policy {
	return Policy{
		Version:     Version,
		Scope:       ScopeDefaultBranchOnly,
		Destructive: DestructiveRequireForce,
	}
}

// Normalize applies the secure version-1 defaults to omitted fields and rejects
// values that the current runtime cannot enforce.
func (p Policy) Normalize() (Policy, error) {
	if p.Version == 0 {
		p.Version = Version
	}
	if p.Scope == "" {
		p.Scope = ScopeDefaultBranchOnly
	}
	if p.Destructive == "" {
		p.Destructive = DestructiveRequireForce
	}
	if p.Version != Version {
		return Policy{}, fmt.Errorf("unsupported ref policy version %d", p.Version)
	}
	if p.Scope != ScopeDefaultBranchOnly {
		return Policy{}, fmt.Errorf("unsupported ref policy scope %q: only %q is supported", p.Scope, ScopeDefaultBranchOnly)
	}
	if p.Destructive != DestructiveRequireForce {
		return Policy{}, fmt.Errorf("unsupported destructive ref policy %q: only %q is supported", p.Destructive, DestructiveRequireForce)
	}
	return p, nil
}

func (p Policy) Decide(relationship Relationship) (Decision, error) {
	if _, err := p.Normalize(); err != nil {
		return Decision{}, err
	}
	switch relationship {
	case RelationshipEqual:
		return Decision{}, nil
	case RelationshipBehind:
		return Decision{Action: true, Reason: "mirror is behind"}, nil
	case RelationshipAhead:
		return Decision{Action: true, Force: true, Reason: "mirror is ahead"}, nil
	case RelationshipDiverged:
		return Decision{Action: true, Force: true, Reason: "mirror is diverged"}, nil
	default:
		return Decision{}, fmt.Errorf("unsupported repository relationship %q", relationship)
	}
}

func (p Policy) ValidateDefaultBranches(source, target, canonicalDefault, mirrorDefault string) error {
	if _, err := p.Normalize(); err != nil {
		return err
	}
	source = strings.TrimSpace(source)
	target = strings.TrimSpace(target)
	canonicalDefault = strings.TrimSpace(canonicalDefault)
	mirrorDefault = strings.TrimSpace(mirrorDefault)
	if mirrorDefault == "" {
		mirrorDefault = canonicalDefault
	}
	if canonicalDefault == "" || mirrorDefault == "" {
		return fmt.Errorf("default branch evidence is incomplete")
	}
	if source != canonicalDefault || target != mirrorDefault {
		return fmt.Errorf("action targets %s/%s but current default branches are %s/%s", source, target, canonicalDefault, mirrorDefault)
	}
	return nil
}
