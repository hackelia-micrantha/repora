# ADR-0009: Scope Boundary (v0.1 vs Future)

Status: Draft

## Decision

Repora v0.1 is limited to Git topology control. Content templating, CI/CD
enforcement, and artifact management are explicitly deferred.

## Rationale

Combining topology reconciliation with content and workflow management
introduces orthogonal complexity:

- File-level diffs vs ref-level diffs
- Provider-specific CI semantics
- Artifact lifecycle management

Separating concerns preserves correctness and enables incremental evolution.

## Consequences

- v0.1 remains small and verifiable
- Future features require new schema versions and ADRs
- CLI surface must expand carefully to avoid ambiguity
