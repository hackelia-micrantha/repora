# ADR-0013: Multi-mirror status before mutation

Status: Implemented

Decision date: 2026-08-02

Last reviewed: 2026-08-02

Supersedes: none

Superseded by: none

Implemented by: PR #76

Related issues: #13, #15

## Context

Repora's topology schema used a mirror list while runtime observation and mutation assumed `mirrors[0]`. Expanding apply directly would combine target identity, artifact migration, partial failure, and journal changes in one unsafe step.

A read-side model is required first so stable target identity, failure isolation, output contracts, and aggregate status semantics can be tested without increasing mutation blast radius.

## Decision

Support multiple mirrors in `repoctl status` before supporting them in plan/apply/sync.

- Canonical setup and commit evidence are shared once per repository.
- Mirrors are observed independently in configuration order.
- A mirror-specific failure remains attached to that mirror and does not hide later results.
- Stable mirror identity is `provider:path`.
- Multiple mirrors require provider/path configuration and duplicate targets are rejected.
- Status JSON advances to version 2.
- Plan/apply/sync reject multiple mirrors before reconciliation observation.

## Alternatives

### Expand status and apply together

Rejected because read contract, artifact identity, execution continuation, and journal semantics require separate review.

### Continue using the first mirror implicitly

Rejected because list position is not durable identity and silently ignores declared topology.

### Use resolved URLs or Git aliases as target identity

Rejected because transport can change and may expose irrelevant or sensitive details.

### Observe mirrors concurrently within one repository

Deferred. Sequential observation is simpler and deterministic; repository entries already have bounded concurrency.

## Consequences

- Status accurately represents all configured mirrors.
- Existing single-mirror plan/apply behavior remains unchanged.
- Status consumers must migrate to the v2 envelope.
- A multi-mirror configuration can be inspected but cannot yet be reconciled.
- Issue #15 must define exact artifact identity and per-mirror evidence before the mutation gate is removed.

## Failure semantics

- canonical/cache failure is repository-level;
- mirror failure produces `ERROR` state and exit `1` while later mirrors remain visible;
- complete ahead/diverged observation produces exit `2`;
- operational error takes precedence over unsafe-state exit reporting.

## Security implications

- Read expansion cannot accidentally widen mutation.
- Stable selectors exclude credentials and transport URLs.
- Ambiguous legacy multi-mirror topology fails configuration loading.
- The future mutation implementation must not treat configuration order as authority.

## Validation

- multiple provider/path mirror configuration tests;
- duplicate and legacy multi-mirror rejection tests;
- shared canonical and ordered mirror observation tests;
- mirror-failure isolation tests;
- command-level v2 output and exit-code tests;
- pre-observation mutation-gate tests.
