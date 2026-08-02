# Architecture decisions

This directory is the index and lifecycle policy for durable Repora architecture decisions.

The original ADR set remains under `docs/rfcs/0001-repora-documentation-set-v0.1/decisions/`. Those records preserve useful reasoning but were created as part of a draft RFC and must not be treated as accepted current behavior merely because they exist.

## Decision lifecycle

Every new or materially revised ADR must declare one status:

- `Proposed` — under review and not implementation authority;
- `Accepted` — the intended durable decision;
- `Implemented` — accepted and represented by merged source/tests;
- `Superseded` — replaced by a named later decision;
- `Rejected` — considered and intentionally not adopted;
- `Historical` — retained for context but no longer current guidance.

An ADR should also include:

```text
Status:
Decision date:
Last reviewed:
Supersedes:
Superseded by:
Implemented by:
Related issues:
```

`Implemented` does not mean every future extension described by an ADR exists. The record must distinguish the accepted invariant from optional future directions.

## Authority rules

- Source and tests define current implemented behavior.
- [`../architecture/current-system.md`](../architecture/current-system.md) explains the merged architecture.
- An accepted ADR explains why a durable choice was made.
- [`../plans/current.md`](../plans/current.md) owns implementation order and release gates.
- GitHub issues own work state and acceptance criteria.
- A proposal must not be described as current behavior in README or architecture documents.

When an ADR and source disagree, either the source is defective or the ADR is stale. The conflict must be resolved explicitly; do not maintain both as valid alternatives.

## Legacy ADR index

| Record | Subject | Current handling |
| --- | --- | --- |
| [ADR-0001](../rfcs/0001-repora-documentation-set-v0.1/decisions/0001-git-as-primary-state-authority.md) | Git as primary state authority | Retained draft; principle remains relevant and should be revalidated when edited. |
| [ADR-0002](../rfcs/0001-repora-documentation-set-v0.1/decisions/0002-unidirectional-canonical-to-mirror-synchronization.md) | Canonical-to-mirror synchronization | Retained draft; current runtime is unidirectional for one default branch. |
| [ADR-0003](../rfcs/0001-repora-documentation-set-v0.1/decisions/0003-divergence-handling.md) | Divergence handling | Retained draft; current implementation adds explicit force gating, stale preflight, and force-with-lease. |
| [ADR-0004](../rfcs/0001-repora-documentation-set-v0.1/decisions/0004-existing-repositories-only-v0.1.md) | Existing repositories only | Retained draft; still matches the current product boundary. |
| [ADR-0005](../rfcs/0001-repora-documentation-set-v0.1/decisions/0005-authentication-model.md) | Authentication model | Retained draft; system Git still owns credentials. |
| [ADR-0006](../rfcs/0001-repora-documentation-set-v0.1/decisions/0006-storage-model.md) | Storage model | Retained draft; requires review against durable `uid`, cache safety, and execution journals. |
| [ADR-0007](../rfcs/0001-repora-documentation-set-v0.1/decisions/0007-concurrency-model.md) | Concurrency model | Retained draft; repository-level bounded concurrency remains implemented. |
| [ADR-0008](../rfcs/0001-repora-documentation-set-v0.1/decisions/0008-disk-usage-optimization.md) | Disk usage optimization | Retained draft; not a current release gate. |
| [ADR-0009](../rfcs/0001-repora-documentation-set-v0.1/decisions/0009-scope-boundary-v0.1-vs-future.md) | v0.1 scope boundary | Retained draft; use the current plan for active scope and deferrals. |
| [ADR-0010](../rfcs/0001-repora-documentation-set-v0.1/decisions/0010-unified-diff-model.md) | Unified diff model | Partially implemented for versioned Git-ref plan artifacts; its universal cross-domain abstraction requires revision before acceptance. |

## Required decision work

### Narrow the plan-artifact decision

ADR-0010 currently mixes two choices:

1. a useful, implemented choice: a versioned artifact is the planner/executor boundary;
2. an unproven future choice: refs, files, workflows, and artifacts should share one universal state/diff model.

The first should be accepted independently. The second should remain proposed until multiple implemented domains demonstrate genuinely shared semantics.

The preferred boundary is a shared envelope with domain-specific tagged actions, not a universal diff object imposed before those domains exist.

### Record explicit ref policy

Before synchronization expands beyond the current default branch, an accepted decision must define:

- branch allowlist semantics;
- tag behavior;
- protected refs;
- wildcard limitations;
- force authorization;
- planner and executor enforcement responsibilities;
- safe defaults.

### Record execution evidence policy

Before journals become required, an accepted decision must define:

- pre-mutation intent persistence;
- final-result persistence;
- fail-closed write behavior;
- execution ID ownership;
- retention and cleanup;
- safe references and redaction;
- whether timestamps belong inside or outside deterministic evidence.

## ADR writing guidance

A useful ADR is narrow and falsifiable. It should contain:

1. **Context** — the concrete problem and constraints;
2. **Decision** — one durable choice;
3. **Alternatives** — credible options that were rejected;
4. **Consequences** — operational costs and limitations;
5. **Security implications** — authority, trust, credentials, mutation, and recovery effects;
6. **Compatibility** — migration and versioning effects;
7. **Implementation boundary** — what is required now versus explicitly deferred;
8. **Validation** — tests or evidence that demonstrate implementation.

Do not use an ADR as a backlog, broad product vision, or substitute for an implementation plan.
