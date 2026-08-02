# ADR-0011: Fail-Closed Immutable Execution Evidence

Status: Accepted

Decision date: 2026-08-02

Last reviewed: 2026-08-02

Supersedes: the unimplemented single-record execution-journal lifecycle

Implemented by: execution-record v2 and audited apply integration under issues #1 and #6

Related issues: #1, #3, #6, #8

## Context

Repora mutates durable Git state. A journal written only after execution cannot prove that an operator reviewed a specific artifact before mutation, while one immutable record written before execution cannot also contain final outcomes without overwrite.

The existing append-only writer used one filename per repository and execution ID. That model was suitable for a standalone record but could not represent both fail-closed pre-mutation intent and final applied, stale, failed, or skipped evidence.

Repora remains local-first and should not add a database, hosted audit service, signing infrastructure, or mutable event store to solve this boundary.

## Decision

One logical repository execution uses two immutable version 2 records sharing one execution ID:

```text
<uid>--<execution-id>--intent.json
<uid>--<execution-id>--result.json
```

The records share the exact plan-artifact digest, repository UID, mode, action ordering, and before/desired state.

### Intent entry

The `INTENT` entry contains only `PLANNED` outcomes. It is persisted after configuration, artifact, current-state, force, and default-branch validation, but before executor stale-ref preflight can reach mutation.

Required intent persistence is fail-closed. If the intent entry cannot be published, no mutation occurs.

### Result entry

The `RESULT` entry is attempted after dry-run preflight or real execution:

- successful dry-run actions are `VALIDATED`;
- successful mutations are `APPLIED` with resulting OIDs;
- stale actions are `STALE`;
- runtime failures are `FAILED`;
- unattempted actions are `SKIPPED`.

A result-write failure returns a nonzero process status even if mutation completed. The intent reference remains available, and operators reconcile current Git state before retry.

## Execution identity

The CLI generates one cryptographically random command-level execution ID. Each selected repository uses that ID with its durable UID, avoiding collisions while correlating one command across repositories.

Execution IDs are evidence correlation identifiers, not authorization tokens or replay keys.

## Storage root

CLI journals are stored beneath the directory containing the loaded configuration file:

```text
<config-directory>/.repora/journal/
```

This provides deterministic local ownership without adding a second configuration source. References exposed by the CLI remain relative to that root.

## Compatibility

- execution-record version 2 adds explicit `INTENT` and `RESULT` phases;
- legacy version 1 records remain parseable and retain their original filename convention;
- new writes use version 2;
- `repora.apply` advances to version 2 to expose execution ID and journal references;
- the version 1 apply schema remains committed for historical consumers.

## Security implications

- records reference the exact artifact by SHA-256 digest;
- transport URLs, credentials, command lines, environments, and absolute cache paths are excluded;
- unsafe diagnostics are replaced with a stable redacted message;
- append-only no-replace publication prevents silent evidence rewriting;
- restrictive permissions and symlink-component rejection remain required;
- records are evidence only and never authority to replay stale mutations.

## Consequences

### Positive

- every CLI dry-run and mutation has durable pre/post evidence;
- a missing result is distinguishable from a completed result;
- journal failure cannot be mistaken for command success;
- intent and result remain independently immutable;
- the design stays filesystem-local and operationally inspectable.

### Costs

- each repository execution writes two files;
- result durability can be uncertain after a successful mutation, requiring operator reconciliation;
- apply JSON requires a versioned migration;
- retention remains an operator responsibility.

## Rejected alternatives

### Mutable single record

Rejected because overwriting intent weakens append-only evidence and creates crash windows around replacement.

### Result-only journaling

Rejected because mutation could occur without durable evidence of the exact approved intent.

### Database or remote audit service

Rejected for the current local-first scope due operational and trust complexity.

### Automatic replay from journal

Rejected because recorded evidence may be stale. Recovery always re-observes and re-plans current state.

## Deferred decisions

- retention automation and indexing;
- cryptographic signing or external provenance;
- remote replication of journal entries;
- approval identities and policy attestations;
- independent provider-side post-push verification.
