# Repora v0.1 Implementation Plan and Checklist

**Date**: May 3, 2026  
**Last Updated**: May 6, 2026  
**Target Release**: Q3 2026  
**Current Stage**: v0.1-alpha

This document now tracks actual implementation state instead of purely planned work.

---

# Current Capability Snapshot

Repora currently supports:

- YAML-based declarative repository specifications
- Concurrent multi-repository status checks
- Divergence detection using Git graph comparison
- Human-readable and JSON output modes
- `plan` command with structured actions
- `apply` command for canonical-to-mirror reconciliation
- Dry-run execution support
- Explicit destructive-operation gating
- Lease-based destructive reconciliation semantics
- Corrupted mirror cache recovery
- Integration testing with real local Git repositories
- CI validation for formatting, vetting, tests, and builds

Current constraints remain intentional:

- existing repositories only
- mirror mode only
- default branch reconciliation only
- unidirectional synchronization only

---

# Phase Status

## Phase 1 — Status and Config

Completed:

- YAML parsing using `gopkg.in/yaml.v3`
- Config validation
- Concurrent repository checks
- Divergence detection
- Human-readable output
- JSON output
- Git timeouts
- Corrupted cache recovery
- Remote HEAD resolution

Remaining:

- structured logging
- expanded error taxonomy
- additional validation edge cases

---

## Phase 2 — Multi-Repository Support

Completed:

- concurrent execution
- ordered aggregation
- partial failure handling
- JSON aggregation
- provider validation
- explicit defaults

Remaining:

- benchmark suites
- advanced branch mapping

---

## Phase 3 — Planning

Completed:

- `plan` command
- structured plan output
- dry-run support
- per-repository planning

Remaining:

- unified diff execution model
- persisted execution plans
- serialized review artifacts

---

## Phase 4 — Apply and Safety

Completed:

- `apply` command
- safe push execution
- divergence blocking
- explicit force gating
- lease-based destructive reconciliation
- integration testing with real repositories

Remaining:

- execution journaling
- audit records
- ref policy configuration
- auth abstraction layer
- progress reporting

---

## Phase 5 — CI and Release Hardening

Completed:

- GitHub Actions CI
- cross-platform builds
- formatting validation
- `go vet`
- automated tests

Remaining:

- release automation
- changelog generation
- binary publishing
- benchmark automation
- security scanning

---

# Recommended Near-Term Milestones

## v0.1-alpha

Goals:

- stable status/plan/apply loop
- integration-tested reconciliation
- lease-based mutation safety
- CI-backed validation

## v0.1-beta

Planned additions:

- README templating
- branch/ref policy model
- execution journaling
- structured audit output
- improved schema definitions

## v0.1

Planned additions:

- stable JSON contracts
- reproducible builds
- hardened error taxonomy
- release packaging
- expanded provider validation

---

# Current Architectural Constraints

Deliberately deferred:

- plugin execution
- arbitrary workflow mutation
- recursive repository creation
- wildcard multi-branch synchronization
- container registry mutation
- hosted control plane behavior

These are intentionally postponed until the mutation model and governance boundaries stabilize.

---

# Major Remaining Risks

## Ref Semantics

Future work should explicitly model:

- branch allowlists
- protected refs
- tag reconciliation policy
- force semantics per ref class

## Auditability

Future work should add:

- execution IDs
- before/after OIDs
- structured action logs
- approval metadata

## Performance

Future work should add:

- benchmark suites
- cache eviction policies
- repository size controls
- memory profiling

---

# Tracking

Recommended:

- GitHub issues for implementation slices
- ADRs for mutation semantics
- RFCs for architectural expansion
- integration-test-first development for reconciliation changes
