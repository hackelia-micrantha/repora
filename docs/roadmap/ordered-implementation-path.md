# Ordered Implementation Path

## Purpose

This roadmap records the dependency order for Repora's current backlog after durable repository identity (`uid`), mirror workflow semantics, and the first runtime transport resolver slice landed.

Feature issues remain authoritative for detailed requirements and acceptance criteria. This document prevents implementation from bypassing architectural or safety prerequisites.

## Product boundary

The first meaningful Repora release is a local-first Git mirror controller that:

- resolves declarative repository topology
- observes source and target refs
- creates a deterministic, versioned plan
- evaluates explicit synchronization policy
- rejects stale execution
- applies authorized mutations with leases
- writes durable evidence

Broader artifact management, document routing, assessment, and hosted-control-plane work follows this substrate.

## Dependency path

```text
DONE #21 durable uid identity
  -> DONE #29 mirror workflow semantics
  -> DONE #16 runtime transport resolver slice
  -> #19 provider/path topology documentation
  -> #22 topology/observation/planning/execution separation
  -> #3 versioned JSON contracts
  -> #8 serialized plan and unified diff execution boundary
  -> #1 filesystem execution journal
  -> #4 ref-policy RFC
  -> #2 ref-policy validation and enforcement
  -> #13 multi-mirror status
  -> #15 multi-mirror apply
  -> #30 optional Anthesis policy integration
  -> #10 security and supply-chain CI
  -> #5 distributable binaries
  -> #11 v0.1 release hardening
```

Issue #16 remains open for the broader configurable-base and transport-selection work. Its prerequisite runtime resolver boundary is complete; #19 documents the currently implemented provider/path behavior without claiming those remaining features.

## Priority bands

### P0 — Semantic and documentation truth

- DONE #29 mirror workflow semantics
- keep README claims aligned with implemented behavior
- maintain this ordered roadmap
- ensure document routing includes the actual Go source roots

Exit condition: source ownership, classifications, destructive-change defaults, stale-plan semantics, and evidence requirements are explicit.

### P1 — Topology foundation

- DONE #16 runtime transport resolver slice
- #19 provider/path topology documentation
- remaining #16 configurable provider bases and transport selection

Exit condition: provider/path is authoritative, runtime URLs are derived, legacy URL compatibility is bounded, and credentials cannot enter serialized topology.

### P2 — One decision path

- #22 planner/executor separation
- #3 JSON contracts
- #8 serialized plan artifact

Exit condition: dry-run and apply consume the same versioned plan and apply does not reconstruct intent.

### P3 — Evidence and policy

- #1 execution journal
- #4 ref-policy RFC
- #2 ref-policy enforcement

Exit condition: every attempted mutation has durable evidence and anything beyond the conservative default branch policy fails closed.

### P4 — Scale and external policy

- #13 multi-mirror status
- #15 multi-mirror apply
- #30 Anthesis integration

Exit condition: mirrors are evaluated independently, partial success is explicit, and optional external policy decisions are deterministic and enforceable.

### P5 — Release readiness

- #10 security CI
- #5 release binaries
- #11 remaining v0.1 hardening

Exit condition: tagged cross-platform binaries, checksums, smoke tests, documented compatibility, low-noise security checks, and a release checklist exist.

## Deferred tracks

These are valid product directions but are not on the core mirror-controller critical path:

- managed repository artifacts (#7, #12)
- advanced document routing (#9, #14, #17, #18, #20, #23)
- repository assessment and evidence reports (#24)

They should reuse the versioned plan, policy, execution, and evidence model rather than introducing separate mutation paths.

## Cross-cutting requirements

Every implementation slice must preserve:

- durable identity through `uid`
- deterministic output for identical semantic inputs
- explicit and bounded mutation scope
- fail-closed destructive behavior
- no secrets or credential-bearing URLs in artifacts
- per-mirror error isolation
- compatibility tests for public machine-readable output
- documented current limitations

## Maintenance rule

Update this document when:

- a dependency is completed or superseded
- an issue is split or reordered
- a new release gate is introduced
- implementation reveals a safety prerequisite

Do not mark an item complete merely because design exists; completion requires the issue's acceptance criteria and merged implementation where applicable.
