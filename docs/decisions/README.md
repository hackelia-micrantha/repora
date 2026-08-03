# Architecture decisions

This directory is the index and lifecycle policy for durable Repora architecture decisions.

The original ADR set remains under `docs/rfcs/0001-repora-documentation-set-v0.1/decisions/`. Those records preserve useful reasoning but must not be treated as current behavior merely because they exist.

## Decision lifecycle

Every new or materially revised ADR must declare one status:

- `Proposed` — under review and not implementation authority;
- `Accepted` — the intended durable decision;
- `Implemented` — accepted and represented by merged source/tests;
- `Superseded` — replaced by a named later decision;
- `Rejected` — considered and intentionally not adopted;
- `Historical` — retained for context but no longer current guidance.

An ADR should include status, decision and review dates, supersession information, implementation references, and related issues.

## Authority rules

- Source and tests define current implemented behavior.
- [`../architecture/current-system.md`](../architecture/current-system.md) explains the merged architecture.
- An accepted ADR explains why a durable choice was made.
- [`../plans/current.md`](../plans/current.md) owns implementation order and release gates.
- GitHub issues own work state and acceptance criteria.
- A proposal must not be described as current behavior in README or architecture documents.

When an ADR and source disagree, either the source is defective or the ADR is stale. The conflict must be resolved explicitly.

## ADR index

| Record | Subject | Current handling |
| --- | --- | --- |
| [ADR-0001](../rfcs/0001-repora-documentation-set-v0.1/decisions/0001-git-as-primary-state-authority.md) | Git as primary state authority | Historical draft; principle remains relevant. |
| [ADR-0002](../rfcs/0001-repora-documentation-set-v0.1/decisions/0002-unidirectional-canonical-to-mirror-synchronization.md) | Canonical-to-mirror synchronization | Historical draft; current runtime remains unidirectional. |
| [ADR-0003](../rfcs/0001-repora-documentation-set-v0.1/decisions/0003-divergence-handling.md) | Divergence handling | Historical draft; current runtime adds explicit force gating, stale preflight, and force-with-lease. |
| [ADR-0004](../rfcs/0001-repora-documentation-set-v0.1/decisions/0004-existing-repositories-only-v0.1.md) | Existing repositories only | Historical draft; still matches the current boundary. |
| [ADR-0005](../rfcs/0001-repora-documentation-set-v0.1/decisions/0005-authentication-model.md) | Authentication model | Historical draft; system Git owns credentials. |
| [ADR-0006](../rfcs/0001-repora-documentation-set-v0.1/decisions/0006-storage-model.md) | Storage model | Historical draft; current implementation adds durable UID cache and execution journals. |
| [ADR-0007](../rfcs/0001-repora-documentation-set-v0.1/decisions/0007-concurrency-model.md) | Concurrency model | Historical draft; repository-level bounded concurrency remains implemented. |
| [ADR-0008](../rfcs/0001-repora-documentation-set-v0.1/decisions/0008-disk-usage-optimization.md) | Disk usage optimization | Historical draft; not a current release gate. |
| [ADR-0009](../rfcs/0001-repora-documentation-set-v0.1/decisions/0009-scope-boundary-v0.1-vs-future.md) | v0.1 scope boundary | Historical draft; use the current plan for active scope. |
| [ADR-0010](../rfcs/0001-repora-documentation-set-v0.1/decisions/0010-unified-diff-model.md) | Versioned domain-specific plan artifacts | Accepted and implemented for exact Git-ref artifacts; the universal cross-domain abstraction is rejected pending evidence. |
| [ADR-0011](0011-fail-closed-execution-evidence.md) | Fail-closed immutable execution evidence | Implemented by PR #74. |
| [ADR-0012](0012-closed-ref-policy-v1.md) | Closed reference synchronization policy v1 | Implemented by PR #75. |

## Future decision gates

Before multi-mirror mutation, an accepted decision must define stable mirror target identity, exact-artifact compatibility, per-mirror result/evidence semantics, and non-atomic recovery.

Before reference scope expands beyond policy v1, a new decision must define branch/tag eligibility, protected refs, wildcard behavior, artifact binding, and compatibility. Existing v1 artifacts must not be reinterpreted under broader policy.

## ADR writing guidance

A useful ADR is narrow and falsifiable. It should contain:

1. **Context** — the concrete problem and constraints;
2. **Decision** — one durable choice;
3. **Alternatives** — credible options that were rejected;
4. **Consequences** — operational costs and limitations;
5. **Security implications** — authority, credentials, mutation, and recovery effects;
6. **Compatibility** — migration and versioning effects;
7. **Implementation boundary** — required now versus deferred;
8. **Validation** — tests or evidence demonstrating implementation.

Do not use an ADR as a backlog, broad product vision, or substitute for an implementation plan.
