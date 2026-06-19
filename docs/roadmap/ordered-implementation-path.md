# Ordered Implementation Path

This document implements GitHub issue #27 as a durable repository artifact. It records the current dependency order for Repora’s backlog after completion of durable repository identity work in #21 / PR #26.

It is intentionally an ordering and scope document, not a replacement for the child issues. The GitHub issues remain the source of detailed requirements, subtasks, and acceptance criteria.

## Purpose

Repora has several interdependent tracks: topology, planning/execution, mutation safety, document routing, managed artifacts, release readiness, and repository evidence reports. Implementing them out of order risks coupling durable state to unstable identities, widening mutation scope before safety controls exist, or drifting Repora into a generic AI orchestration system.

This roadmap keeps implementation sequenced through stable architectural boundaries.

## Core topology and execution foundation

```text
DONE #21 / PR #26 durable uid identity
  -> #16 transport resolver
  -> #19 topology docs
  -> #22 topology/planning/execution split
  -> #3 JSON/schema contracts
  -> #8 unified diff plan artifact
  -> #6/#1 journaling
  -> #4/#2 branch/ref policy
  -> #13 multi-mirror status
  -> #15 multi-mirror apply
```

### P0: next architectural gates

- [ ] #16 Architecture: introduce transport resolver layer
- [ ] #19 Docs: document provider/path topology schema v1
- [ ] #22 Architecture: separate topology, planning, and execution layers
- [ ] #3 Stabilize CLI JSON contracts and publish schema definitions

These should land before additional mutation, artifact-generation, or multi-mirror behavior expands. #16 and #19 complete the identity/location/transport model. #22 defines the runtime ownership split. #3 makes machine-readable output safe for automation and durable artifacts.

### P1: mutation safety and auditability

- [ ] #8 Implement ADR-0010 unified diff execution model
- [ ] #6 Epic: execution journaling and audit trail for repository mutations
- [ ] #1 Implement filesystem execution journal writer and audit record output
- [ ] #4 RFC: explicit branch and ref synchronization policy model
- [ ] #2 Implement branch and ref policy validation

These issues define how Repora moves from intent to reviewable mutation. The ordering matters: the planner/executor boundary and JSON contracts should exist before plan artifacts and journals become public contracts.

### P2: multi-mirror runtime

- [ ] #13 Runtime: support multi-mirror status results
- [ ] #15 Runtime: support multi-mirror apply and reconciliation

Read-side multi-mirror status must land before multi-mirror apply. Apply expands mutation scope and should not precede planner/executor separation, branch/ref policy, or audit behavior.

## Document routing governance

```text
#23 trust tiers
  -> #18 deterministic route tests
  -> #9 subsystem route manifests
  -> #20 hierarchical summaries
  -> #17 context receipts
  -> #14 AST-aware routing
```

### P3: routed-context governance

- [ ] #23 Add trust tiers for document routing
- [ ] #18 Add deterministic route tests
- [ ] #9 Add per-subsystem document route manifests
- [ ] #20 Add hierarchical summaries for routed context
- [ ] #17 Add context receipts for routed retrieval
- [ ] #14 Add AST-aware source routing

Routing is part of the AI trust boundary. Trust tiers and deterministic tests must come before richer routing features are treated as reliable. AST-aware routing should extend deterministic routing; it should not become opaque semantic retrieval.

## Managed artifacts

```text
#22/#3/#8 execution boundary
  -> #7 managed artifact model
  -> #12 README templating as first managed artifact
```

### P4: managed artifacts

- [ ] #7 Epic: managed repository artifact generation model
- [ ] #12 Implement README templating as the first managed artifact type

Managed artifacts must stay bounded. README templating is acceptable as the first slice because it is deterministic, text-based, diffable, and low risk. This track must not expand into arbitrary file generation or plugin execution without a new explicit design issue.

## Release readiness

```text
#3 JSON/schema contracts
  -> #10 security and supply-chain CI
  -> #5 release binaries
  -> #11 v0.1 release-hardening checklist
```

### P4: release readiness

- [ ] #10 Add security scanning and supply-chain validation in CI
- [ ] #5 Add release automation and distributable binaries
- [ ] #11 Epic: release hardening for v0.1

Release work can proceed in parallel with care, but release documentation should describe v0.1 as early, local-first, and conservative around mutation support. Shipping binaries must not imply production-grade mutation semantics.

## Assessment and evidence

```text
#3 JSON/schema contracts
  -> #24 repository assessment and evidence framework
```

### P5: assessment and evidence

- [ ] #24 Add repository assessment and evidence framework

Assessment and evidence reports are strategically useful, but broad. They should remain versioned repository reports that reference GitHub issues, PRs, commits, and files. They must not duplicate project-management state or become an autonomous workflow/control-plane layer.

## Scope boundaries

### In scope

- Local-first repository topology and evidence management
- Durable `uid` identity for persisted artifacts
- Provider/path topology and runtime transport resolution
- Deterministic planner/executor boundaries
- Versioned machine-readable JSON contracts
- Diff-first plan artifacts
- Local execution journals
- Conservative branch/ref policy
- Multi-mirror status before multi-mirror mutation
- Trust-aware deterministic document routing
- Versioned reports and evidence artifacts

### Out of scope

- Generic AI agent runner behavior
- Anthesis-style approval or governance runtime
- Remote control plane
- Autonomous workflow orchestration
- Arbitrary file generation or plugin execution
- Semantic RAG as the primary routing model
- Full rollback automation for v0.1
- Historical audit database
- Production-grade release claims

## Cross-cutting requirements

- Durable artifacts must include `kind` and `version` where machine-readable consumers rely on them.
- Durable artifacts must include repository `uid` where repository correlation matters.
- Human output may use `id`, but persistent state must not key only by `id`, provider/path, or derived URL.
- Derived transport URLs are runtime state, not durable identity.
- Dry-run and apply should share the same planning path.
- Mutation-related issues must define failure behavior, partial failure behavior, audit behavior, and JSON output behavior.
- Routing-related issues must define trust, determinism, testability, and receipt behavior where applicable.
- Reports and receipts must include redaction guidance for local paths, URLs, diffs, snippets, and errors.
- Release docs must describe v0.1 as early, local-first, and conservative rather than production-grade.

## Close conditions for #27

Issue #27 can close when one of these is true:

1. The ordered path has been completed through the required child issues.
2. The ordered path is explicitly superseded by a newer release milestone or architecture roadmap.
3. The backlog is reorganized so this document is no longer the source of implementation order.

Before closing, confirm:

- [ ] #16, #19, #22, #3, and #8 are complete or superseded.
- [ ] #13 lands before #15.
- [ ] Routing governance starts with #23 and #18.
- [ ] #7/#12 remain bounded to managed, deterministic, diffable artifacts.
- [ ] #24 remains bounded to repository reports, evidence, scorecards, and schemas.
- [ ] No child issue drifts into Anthesis/governance/control-plane semantics without a new explicit issue.

## Maintenance rules

- Update this document when child issues close or are superseded.
- Do not duplicate every child issue requirement here.
- Do not close parent epics automatically when one child slice lands.
- Do not block small bug fixes or CI fixes that are unrelated to this ordered path.
