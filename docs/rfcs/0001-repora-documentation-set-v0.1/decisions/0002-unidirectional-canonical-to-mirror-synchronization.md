# ADR-0002: Unidirectional Canonical-to-Mirror Synchronization

Status: Draft

## Decision

Repora v0.1 shall support only unidirectional synchronization from canonical
remotes to mirror remotes using full mirror semantics.

## Rationale

Bidirectional synchronization introduces ambiguity around conflict resolution,
canonicality, history rewriting, and policy ownership. A single authoritative
direction gives Repora a tractable and auditable execution model.

## Consequences

- Mirror-side commits, tags, or ref changes are classified as drift
- Repora does not merge or reconcile independent histories in v0.1
- Canonical state remains the only source from which mirror state may be derived
