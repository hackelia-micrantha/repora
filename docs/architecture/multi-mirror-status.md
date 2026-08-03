# Multi-mirror status architecture

Status: Implemented by PR #76

Repora may observe multiple declared mirrors without enabling multi-mirror mutation.

## Boundary

`repoctl status` evaluates every configured mirror. `repoctl plan`, `repoctl apply`, and `repoctl sync` remain explicitly single-mirror until the exact multi-mirror execution contract is implemented.

This ordering prevents a read-side topology expansion from silently widening mutation scope.

## Observation flow

For each repository entry:

1. resolve and prepare the durable UID cache once;
2. configure, fetch, and resolve canonical once;
3. capture canonical commit evidence once;
4. visit mirrors sequentially in configuration order;
5. configure each mirror through a runtime-only alias such as `mirror-0`;
6. fetch and compare that mirror against `canonical/HEAD`;
7. preserve its commit, state, counts, or local error;
8. continue to later mirrors after a mirror-specific failure.

Repository entries remain concurrently processed through the CLI `--parallel` limit. Mirrors within one repository are intentionally sequential in the first read-side implementation.

## Target identity

The durable mirror selector in status output is:

```text
<provider>:<provider-relative-path>
```

Examples:

```text
github:hackelia-micrantha/repora
gitlab:micrantha/backup/repora
```

Configuration ordering is presentation and execution order, not identity. Runtime Git remote aliases and resolved URLs are transport state and are never durable selectors.

When several mirrors are configured, every mirror must use provider/path form. Bounded single-mirror legacy URL input remains accepted and is projected to a safe repository path when possible; hostnames, credentials, query values, fragments, and transport details are not exposed as identity.

## Failure semantics

Canonical/cache failure is repository-level because no mirror comparison is trustworthy without canonical evidence.

Mirror-specific resolution, configuration, fetch, HEAD, divergence, or commit-evidence failure produces:

- the stable mirror target;
- state `ERROR`;
- an error message scoped to that mirror;
- continued observation of later mirrors.

Aggregate command status is deterministic:

- any incomplete repository or mirror evidence: exit `1`;
- otherwise any `AHEAD` or `DIVERGED` mirror: exit `2`;
- otherwise: exit `0`.

Operational failure takes precedence over unsafe-state reporting because the complete topology was not observed.

## Public contract

`repoctl status --json` emits `repora.status` version 2. Each mirror includes:

- `target`;
- provider/path topology;
- symbolic ref;
- commit evidence;
- state and ahead/behind counts;
- optional error for `ERROR` state.

Status v1 remains committed as a historical schema. Consumers must inspect both `kind` and `version`.

## Mutation gate

Before any reconciliation observation, plan/apply/sync reject a repository with more than one mirror. They do not select the first mirror or reuse status ordering as implicit targeting.

Issue #15 owns the future mutation contract: exact artifact identity, complete preflight, per-mirror outcomes, journal evidence, partial failure, and non-atomic recovery.
