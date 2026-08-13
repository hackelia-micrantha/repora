// Package managedartifact is Repora's authoritative Go implementation of the
// repora.io/managed-artifact-plan v1 contract and the bounded managed README
// lifecycle built on that contract.
//
// Plan, ParsePlan, validation, text-safety rules, size limits, required-field
// semantics, target identity checks, mode/content digest invariants, and strict
// JSON behavior must remain centralized here. Callers should depend on this
// package rather than introducing a second internal model/parser/validator for
// the same serialized contract.
package managedartifact
