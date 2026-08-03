# Current system architecture

Status: Current

This document describes merged Repora behavior. Future architecture belongs in proposals or ADRs until implemented.

## Product boundary

Repora is a local-first Git mirror controller exposed through the `repoctl` Go CLI.

For each repository entry it currently supports:

- one GitLab canonical repository;
- one or more GitHub/GitLab mirrors for status observation;
- exactly one mirror for plan/apply/sync;
- provider-relative `provider + path` topology with bounded single-mirror legacy URL compatibility;
- stable target identity as `provider:path`;
- runtime HTTPS transport resolution;
- default-branch-only closed ref policy;
- independent `EQUAL`, `BEHIND`, `AHEAD`, `DIVERGED`, or `ERROR` status per mirror;
- provider/path-bound exact single-mirror plan artifact v2 export/import;
- historical single-mirror plan artifact v1 import compatibility;
- normal pushes and explicitly authorized lease-protected overwrites;
- fail-closed immutable intent/result journal evidence;
- bounded repository-level concurrency.

It does not provide multi-mirror mutation, tags, non-default branches, deleted-ref reconciliation, provider provisioning, or a hosted control plane.

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

### Plan and apply

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
| `internal/plan` | deterministic single-mirror reconciliation actions and compatibility projection | Git reads/writes or durable serialization |
| `internal/planartifact` | versioned exact artifact parsing, provider-path validation, historical compatibility, and plan conversion | Observation or execution policy |
| `internal/executor` | complete preflight, ordered mutation, and action outcomes | Recomputing status or policy |
| `internal/apply` | artifact construction, configuration binding, authorization, audit integration, and public apply results | Independent reconciliation policy |
| `internal/journal` | immutable intent/result evidence, artifact digest reference, redaction, and append-only local persistence | Mutation or replay authority |
| `internal/git` | bounded Git subprocesses, cache safety, refs, pushes, leases, timeouts, and redaction | Product policy or identity |
| `cmd/repoctl` | command routing, concurrency, status aggregation, mutation gate, artifact I/O, rendering, and exit semantics | Git mechanics or duplicated planning |

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

`status.Check` remains the exact one-mirror observation used by planning.

`status.CheckAll` shares canonical setup and observes each declared mirror sequentially. Mirror-specific failures produce state `ERROR` and remain in output with stable target identity. Later mirrors are still observed.

Aggregate exit status:

- incomplete canonical or mirror evidence: `1`;
- otherwise any ahead/diverged mirror: `2`;
- otherwise: `0`.

Operational failure takes precedence over unsafe-state reporting.

## Plan artifact compatibility

New production plans emit reconciliation artifact version 2. Every ref contains provider, provider-relative path, runtime alias, and branch. Imported v2 artifacts validate canonical and mirror paths against configuration before repository Git reads.

Version 1 remains parseable under its historical single-mirror provider/alias contract. It cannot authorize multi-mirror targeting and is never interpreted as path-bound identity.

The compatibility `repora.plan` v1 output remains provider-oriented and is still a view only.

## Planning and execution safety

Plan/apply/sync reject multi-mirror repositories before Git observation. They never select `mirrors[0]` implicitly.

For an accepted single-mirror repository, execution validates artifact metadata, UID/topology/path binding, state/action/force intent, default branches, and all expected OIDs before action zero. Forced actions additionally use force-with-lease.

`plan --artifact` and `apply --plan-file` share the exact artifact boundary.

## Execution journal

Apply and dry-run write immutable version-2 intent/result records under the configuration directory:

```text
.repora/journal/<uid>--<execution-id>--intent.json
.repora/journal/<uid>--<execution-id>--result.json
```

Intent failure prevents mutation. Result-write failure returns nonzero even after completed mutation. Journals may reference exact plan artifact version 1 or 2 through the serialized artifact digest. Journals are evidence, never replay authority.

## Concurrency and atomicity

Repository entries use bounded concurrency. Mirrors inside one status operation are sequential in configuration order.

There is no cross-repository or cross-remote transaction. Future multi-mirror execution must define continuation and partial outcomes explicitly and must not claim rollback or atomicity.

## Public contracts

Current public envelopes include:

- `repora.status` v2;
- compatibility `repora.plan` v1;
- exact reconciliation plan v2, with v1 historical import support;
- `repora.apply` v2;
- execution record v2.

Historical schemas remain committed. Consumers must inspect `kind` and `version`.

## Immediate architecture gaps

1. multi-action planning from multi-mirror observation;
2. mapping each path-bound reviewed target to its current runtime alias;
3. complete all-target preflight and independent ordered execution;
4. per-mirror apply and journal outcomes with non-atomic recovery;
5. supported v0.1 release packaging and verification.

Managed artifacts, advanced document routing, assessments, and Anthesis integration remain deferred and must reuse the core plan, policy, execution, and evidence boundaries.
