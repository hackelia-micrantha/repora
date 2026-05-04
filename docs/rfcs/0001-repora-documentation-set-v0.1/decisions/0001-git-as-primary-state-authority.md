# ADR-0001: Git as the Primary State Authority

Status: Draft

## Decision

Repora shall use the system `git` executable and the observed state of Git
remotes as its primary authority. Repora v0.1 shall not persist repository state
in an internal database.

## Rationale

Git already provides the object model, ref model, transport semantics, and
history operations needed for v0.1. Introducing a parallel persistence layer
would create reconciliation risk between Repora's internal model and the actual
repository graph.

## Consequences

- `git` must be installed and available in the execution environment
- Repository observation may be slower than querying a local database, but it
  remains closer to the true operational state
- Failure modes remain mostly Git-native and therefore easier for operators to
  diagnose
