# Current system architecture

Status: Current

This document describes merged Repora behavior. Future architecture belongs in proposals or ADRs until implemented.

## Product boundary

Repora is a local-first Git mirror controller exposed through the `repoctl` Go CLI.

For each repository entry it currently supports:

- one GitLab canonical repository;
- one or more GitHub/GitLab mirrors for status, exact planning, and audited dry-run;
- exactly one mirror for real apply/sync mutation;
- provider-relative `provider + path` topology with bounded single-mirror legacy URL compatibility;
- stable target identity as `provider:path`;
- runtime HTTPS transport resolution;
- default-branch-only closed ref policy;
- independent `EQUAL`, `BEHIND`, `AHEAD`, `DIVERGED`, or `ERROR` status per mirror;
- provider/path-bound exact plan artifact v2 export across all required mirror actions;
- historical single-mirror plan artifact v1 import compatibility;
- complete multi-target topology, policy, branch, and OID preflight without mutation;
- normal single-mirror pushes and explicitly authorized lease-protected overwrites;
- fail-closed immutable intent/result journal evidence;
- bounded repository-level concurrency.

It does not provide real multi-mirror mutation, tags, non-default branches, deleted-ref reconciliation, provider provisioning, or a hosted control plane.

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

### Multi-mirror audited dry-run

```text
configuration and exact artifact v2
  -> complete current multi-mirror observation
  -> bind each provider:path target to its current Git remote alias
  -> validate repository UID, topology, state/action/force intent, and default branches
  -> persist one repository-level execution-record v3 INTENT
  -> validate every expected source and target OID before action zero
  -> persist one repository-level execution-record v3 RESULT
  -> render stable provider:path actions and journal references
```

The reviewed artifact remains unchanged. Runtime aliases are local execution details and are never target authority. Dry-run performs no push and does not require force authorization for reviewed destructive intent.

### Real apply and sync

```text
configuration and closed ref policy
  -> explicit exactly-one-mirror gate
  -> observation and classification
  -> path-bound exact artifact
  -> immutable journal intent
  -> complete structural and stale-ref preflight
  -> mutation
  -> immutable journal result
  -> structured output and process status
```

## Package ownership

| Package | Owns | Must not own |
| --- | --- | --- |
| `internal/config` | strict YAML, durable identity, safe endpoint path identity, topology/ref-policy normalization, duplicate target rejection | Runtime URL derivation or Git operations |
| `internal/refpolicy` | closed versioned ref scope and relationship-to-intent decisions | Git operations or CLI authorization parsing |
| `internal/transport` | runtime provider/path URL resolution | Durable identity or policy |
| `internal/status` | single- and multi-mirror observation, target identity, divergence, and commit evidence | Mutation decisions or pushes |
| `internal/plan` | deterministic reconciliation actions and compatibility projection | Git reads/writes or durable serialization |
| `internal/planartifact` | versioned exact artifact parsing, provider-path validation, historical compatibility, and plan conversion | Observation or execution policy |
| `internal/executor` | complete OID preflight, optional runtime alias bindings, ordered mutation, and action outcomes | Recomputing status or policy |
| `internal/apply` | artifact construction, configuration/status/policy binding, audited dry-run orchestration, authorization, and public apply results | Implicit target selection or independent policy |
| `internal/journal` | immutable intent/result evidence, artifact digest reference, path-bound record v3, redaction, and local persistence | Mutation or replay authority |
| `internal/git` | bounded Git subprocesses, cache safety, refs, pushes, leases, timeouts, and redaction | Product policy or identity |
| `cmd/repoctl` | command routing, concurrency, status/plan aggregation, dry-run routing, mutation gate, artifact I/O, rendering, and exit semantics | Git mechanics or duplicated planning |

## Identity and runtime binding

Repora distinguishes:

- `id`: human-facing repository alias;
- `uid`: durable logical repository identity;
- `(provider, path)`: declarative location and stable repository/mirror selector;
- configuration index: deterministic order only;
- resolved URL and Git remote alias: ephemeral transport state.

Status v2, plan artifact v2, and execution-record v3 use provider/path identity. URLs, credentials, local filesystem paths, and array indexes are excluded.

Imported artifacts are matched to configuration by durable UID and provider/path. The executor receives a separate runtime-binding map from stable target identity to current local alias. Artifact aliases remain serialized compatibility/execution detail and cannot retarget an action after mirror reordering.

## Reference policy

Ref-policy version 1 has one interpretation:

- `scope: default-branch-only`;
- `destructive: require-force`.

Omission normalizes to these values. Unsupported expansion fails configuration loading. Planning records destructive intent. Real mutation separately requires `--force`; dry-run may validate forced intent without authorizing mutation.

## Planning and preflight safety

`repoctl plan --artifact` is the authoritative machine-readable multi-mirror plan contract. The legacy `repoctl plan --json` compatibility view remains single-mirror only.

Before audited multi-mirror dry-run writes intent, Repora validates:

- artifact version, kind, repository cardinality, and durable UID;
- configured canonical provider/path;
- every configured mirror target exactly once;
- current status completeness;
- one action exactly when policy requires one;
- force intent against the observed relationship;
- current default branches through runtime-bound aliases.

After intent persistence, executor preflight validates every expected source and target OID. A later stale target leaves earlier actions `SKIPPED`, marks the offending action `STALE`, performs no mutation, and still attempts result persistence.

## Execution journal

Path-bound plan artifact v2 writes execution-record v3 evidence. Version 3 adds provider-relative `path` to each source and target ref. Historical execution-record v1 and v2 remain parseable; their schemas remain committed.

Entries remain:

```text
.repora/journal/<uid>--<execution-id>--intent.json
.repora/journal/<uid>--<execution-id>--result.json
```

One command-level execution ID is shared across selected repositories, while each repository writes its own correlated pair. Intent failure prevents mutation. Result-write failure returns nonzero. Journals are evidence, never replay authority.

## Concurrency and atomicity

Repository entries use bounded concurrency. Mirrors inside one repository are observed, planned, and preflighted deterministically.

There is no cross-repository or cross-remote transaction. Real multi-mirror execution must continue independent later actions after a runtime failure, preserve every target outcome, and make no rollback or atomicity claim.

## Public contracts

Current public envelopes include:

- `repora.status` v2;
- compatibility `repora.plan` v1 for single-mirror topology;
- exact reconciliation plan v2, including multiple actions, with v1 historical import support;
- `repora.apply` v2 for the existing public result view;
- execution record v3, with v1/v2 historical parsing support.

Multi-mirror dry-run human output is supported. Multi-mirror `apply --dry-run --json` is intentionally rejected until a versioned per-target apply output is published.

## Immediate architecture gaps

1. versioned per-target apply output for multi-mirror execution;
2. independent ordered real mutation that continues after one remote fails;
3. path-bound applied/failed/skipped evidence for real multi-mirror mutation;
4. supported v0.1 release packaging and verification.

Managed artifacts, advanced document routing, assessments, and Anthesis integration remain deferred and must reuse the core plan, policy, execution, and evidence boundaries.
