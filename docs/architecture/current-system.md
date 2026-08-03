# Current system architecture

Status: Current

This document describes merged Repora behavior. Future architecture belongs in proposals or ADRs until implemented.

## Product boundary

Repora is a local-first Git mirror controller exposed through the `repoctl` Go CLI.

For each repository entry it currently supports:

- one GitLab canonical repository;
- one or more GitHub/GitLab mirrors for status and exact planning;
- exactly one mirror for apply/sync;
- provider-relative `provider + path` topology with bounded single-mirror legacy URL compatibility;
- stable target identity as `provider:path`;
- runtime HTTPS transport resolution;
- default-branch-only closed ref policy;
- independent `EQUAL`, `BEHIND`, `AHEAD`, `DIVERGED`, or `ERROR` status per mirror;
- provider/path-bound exact plan artifact v2 export across all required mirror actions;
- historical single-mirror plan artifact v1 import compatibility;
- normal single-mirror pushes and explicitly authorized lease-protected overwrites;
- fail-closed immutable intent/result journal evidence for current apply/dry-run;
- bounded repository-level concurrency.

It does not provide multi-mirror apply/sync, multi-mirror audited dry-run, tags, non-default branches, deleted-ref reconciliation, provider provisioning, or a hosted control plane.

## Runtime flows

### Status

```text
configuration
  -> durable cache and canonical observation once
  -> each mirror in configuration order
  -> independent commit/divergence/error result
  -> repora.status v2 human or JSON output
```

A mirror failure does not hide later mirrors. Canonical failure remains repository-level.

### Multi-mirror plan

```text
configuration and closed ref policy
  -> multi-mirror status observation
  -> match results to configuration by provider:path
  -> resolve canonical branch/OID once
  -> resolve required mirror branches/OIDs
  -> ordered path-bound exact artifact v2
  -> human plan or --artifact output
```

Equal mirrors contribute no action. Behind mirrors contribute normal actions. Ahead and diverged mirrors contribute forced intent. Observation array order is not authority; artifact actions follow configuration order after identity matching.

### Apply and sync

```text
configuration and closed ref policy
  -> explicit exactly-one-mirror gate
  -> observation and classification
  -> path-bound exact artifact v2
  -> immutable journal intent
  -> complete structural and stale-ref preflight
  -> mutation or dry-run validation
  -> immutable journal result
  -> structured output and process status
```

## Package ownership

| Package | Owns | Must not own |
| --- | --- | --- |
| `internal/config` | strict YAML, durable identity, safe endpoint path identity, topology/ref-policy normalization, duplicate target rejection | Runtime URL derivation or Git operations |
| `internal/refpolicy` | closed versioned ref scope and relationship-to-intent decisions | Git operations or CLI authorization parsing |
| `internal/transport` | runtime provider/path URL resolution | Durable identity or policy |
| `internal/status` | single-mirror reconciliation observation plus multi-mirror read-side observation, target identity, divergence, and commit evidence | Mutation decisions or pushes |
| `internal/plan` | deterministic reconciliation actions and compatibility projection | Git reads/writes or durable serialization |
| `internal/planartifact` | versioned exact artifact parsing, provider-path validation, historical compatibility, and plan conversion | Observation or execution policy |
| `internal/executor` | complete preflight, ordered single-mirror mutation, and action outcomes | Recomputing status or policy |
| `internal/apply` | single-mirror execution orchestration plus exact multi-mirror artifact construction from status observations | Independent reconciliation policy or implicit target selection |
| `internal/journal` | immutable intent/result evidence, artifact digest reference, redaction, and append-only local persistence | Mutation or replay authority |
| `internal/git` | bounded Git subprocesses, cache safety, refs, pushes, leases, timeouts, and redaction | Product policy or identity |
| `cmd/repoctl` | command routing, concurrency, status aggregation, exact multi-mirror plan aggregation, mutation gate, artifact I/O, rendering, and exit semantics | Git mechanics or duplicated planning |

## Identity and location

Repora distinguishes:

- `id`: human-facing repository alias;
- `uid`: durable logical repository identity;
- `(provider, path)`: declarative location and stable repository/mirror selector;
- configuration index: deterministic order only;
- resolved URL and Git remote alias: ephemeral transport state.

Status v2 and plan artifact v2 use provider/path identity. URLs, credentials, local filesystem paths, and array indexes are excluded. Runtime aliases remain in the artifact only as execution details and are not target authority.

When multiple mirrors are configured, each must use provider/path form and duplicate targets are rejected. Single-mirror legacy URLs remain compatibility input and can be reduced to a safe provider-relative path for new exact plans when supported.

## Reference policy

Ref-policy version 1 has one interpretation:

- `scope: default-branch-only`;
- `destructive: require-force`.

Omission normalizes to these values. Unsupported expansion fails configuration loading. Planning records destructive intent; real mutation separately requires `--force`.

## Status semantics

`status.Check` remains the exact one-mirror observation used by the current apply path.

`status.CheckAll` shares canonical setup and observes each declared mirror sequentially. Mirror-specific failures produce state `ERROR` and remain in output with stable target identity. Later mirrors are still observed.

Aggregate exit status:

- incomplete canonical or mirror evidence: `1`;
- otherwise any ahead/diverged mirror: `2`;
- otherwise: `0`.

Operational failure takes precedence over unsafe-state reporting.

## Exact multi-mirror planning

`apply.BuildRepositoryArtifact` accepts one complete status result and matches each observation to configured topology by target identity. It resolves canonical branch and source OID once, then resolves branches and target OIDs only for mirrors requiring action.

Actions are emitted in configuration order. Missing, duplicate, error-bearing, or identity-mismatched observations suppress exact artifact creation.

`repoctl plan --artifact` is the machine-readable multi-mirror plan contract. Human plan output also shows stable targets. The legacy `repoctl plan --json` compatibility view remains single-mirror only and is rejected for multi-mirror topology rather than silently changing version-1 semantics.

Planning may return exit `2` when the complete artifact contains destructive intent and `--force` was not supplied. The artifact is still emitted for review.

## Plan artifact compatibility

New production plans emit reconciliation artifact version 2. Every ref contains provider, provider-relative path, runtime alias, and branch.

Version 1 remains parseable under its historical single-mirror provider/alias contract. It cannot authorize multi-mirror targeting and is never interpreted as path-bound identity.

## Planning and execution safety

Plan supports multiple mirrors. Apply/sync still reject multi-mirror repositories before Git observation and never select `mirrors[0]` implicitly.

For an accepted single-mirror execution, apply validates artifact metadata, UID/topology/path binding, state/action/force intent, default branches, and all expected OIDs before action zero. Forced actions additionally use force-with-lease.

## Execution journal

Apply and dry-run write immutable version-2 intent/result records under the configuration directory:

```text
.repora/journal/<uid>--<execution-id>--intent.json
.repora/journal/<uid>--<execution-id>--result.json
```

Intent failure prevents mutation. Result-write failure returns nonzero even after completed mutation. Journals may reference exact plan artifact version 1 or 2 through the serialized artifact digest. Journals are evidence, never replay authority.

Multi-mirror plan does not create journal evidence because it does not enter the execution boundary.

## Concurrency and atomicity

Repository entries use bounded concurrency. Mirrors inside status and planning are processed sequentially in configuration order after identity matching.

There is no cross-repository or cross-remote transaction. Future multi-mirror execution must define continuation and partial outcomes explicitly and must not claim rollback or atomicity.

## Public contracts

Current public envelopes include:

- `repora.status` v2;
- compatibility `repora.plan` v1 for single-mirror topology;
- exact reconciliation plan v2, including multiple actions, with v1 historical import support;
- `repora.apply` v2;
- execution record v2.

Historical schemas remain committed. Consumers must inspect `kind` and `version`.

## Immediate architecture gaps

1. mapping each path-bound imported target to its current runtime alias;
2. complete all-target preflight and audited dry-run;
3. independent ordered multi-mirror mutation with continuation after runtime failure;
4. per-mirror apply and journal outcomes with non-atomic recovery;
5. supported v0.1 release packaging and verification.

Managed artifacts, advanced document routing, assessments, and Anthesis integration remain deferred and must reuse the core plan, policy, execution, and evidence boundaries.
