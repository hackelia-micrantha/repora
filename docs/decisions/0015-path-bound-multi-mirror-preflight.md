# ADR-0015: bind multi-mirror execution by provider path before mutation

Status: Implemented

Decision date: 2026-08-02

Review date: 2026-08-02

Implementation: PR #79

Related issues: #15

## Context

Reconciliation artifact v2 can contain several mirror actions and records each reviewed source and target by provider-relative path. The artifact also retains local Git remote aliases, but those aliases are derived from configuration order and may change when mirrors are reordered.

Using serialized aliases or action position as target authority could apply a reviewed action to the wrong remote. Real multi-mirror mutation also requires proof that every action is structurally valid and current before action zero.

Execution record v2 cannot preserve provider/path evidence because its reference shape contains only provider, runtime alias, and branch.

## Decision

Repora will stage multi-mirror execution through an audited dry-run before enabling real mutation.

For each repository:

1. match the artifact repository by durable `uid`;
2. match configured and observed mirrors by `provider:path`;
3. derive current local Git aliases from the current configuration;
4. pass a separate runtime-binding map to the executor;
5. leave artifact actions unchanged;
6. validate topology, status/action/force agreement, and default branches before intent persistence;
7. persist one repository-level intent record;
8. validate every source and target OID before action zero;
9. persist one repository-level result record;
10. perform no mutation in this slice.

Path-bound artifact v2 execution evidence uses execution-record version 3. Version 3 adds provider-relative path to source and target refs. Versions 1 and 2 remain parseable and are not reinterpreted as path-bound evidence.

Real multi-mirror apply and sync remain gated until independent continuation and per-target result semantics are implemented.

## Alternatives considered

### Trust serialized Git aliases

Rejected. Aliases are runtime details derived from configuration order and can change without changing the reviewed target.

### Match actions by array position

Rejected. Configuration order is deterministic ordering, not identity. Reordering must not retarget imported intent.

### Rewrite the artifact with current aliases

Rejected. The reviewed artifact and its digest must remain the execution-intent authority. Runtime binding belongs beside the artifact, not inside a mutated copy.

### Add provider paths to execution-record v2

Rejected. That would change an existing public schema without a version transition.

### Enable real mutation in the same change

Rejected. Audited dry-run provides executable experience with topology, branch, OID, and evidence boundaries before partial real outcomes are introduced.

## Consequences

- imported artifacts survive harmless mirror reordering without retargeting;
- complete all-target preflight occurs before any future action zero;
- a stale later target yields ordered `SKIPPED`/`STALE` evidence and zero mutation;
- path-bound intent and result evidence are machine-readable in record v3;
- current public apply JSON remains v2 and multi-mirror dry-run is human-output only;
- real mutation requires one further versioned result/evidence slice;
- runtime alias derivation remains provider implementation detail.

## Security implications

The decision prevents confused-deputy targeting through stale aliases or mirror position. Topology and policy mismatch fail before journal intent and before repository ref reads where possible. Expected OID mismatch fails after intent but before mutation and is preserved as result evidence.

Provider paths are validated as relative namespace-qualified identifiers and exclude URLs, credentials, filesystem paths, traversal, and unsafe delimiters.

## Compatibility

- reconciliation artifact v2 is required for multi-mirror preflight;
- artifact v1 remains historical single-mirror compatibility only;
- execution-record v1 and v2 remain parseable;
- execution-record v3 is emitted for artifact v2 evidence;
- real single-mirror behavior remains available through the existing path;
- multi-mirror `apply --dry-run --json` is rejected until a new per-target apply contract exists.

## Validation

Implementation must demonstrate:

- mirror reordering changes runtime aliases without changing reviewed targets;
- unknown targets fail before Git ref reads and journal writes;
- every action is preflighted before mutation;
- a stale later target leaves earlier actions skipped and writes path-bound result evidence;
- dry-run performs zero push operations;
- real multi-mirror apply remains gated;
- formatting, race tests, integration tests, cross-platform builds, vulnerability scanning, and CodeQL pass.
