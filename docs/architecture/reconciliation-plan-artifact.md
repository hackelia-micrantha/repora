# Reconciliation plan artifact

Status: Current

Repora uses a versioned reconciliation plan artifact as the review and execution boundary for repository mutations.

New plans emit `repora.io/reconciliation-plan` version `2`, described by `schemas/reconciliation-plan-v2.schema.json`. Version 1 remains parseable for historical single-mirror compatibility.

## Operator workflow

Export the exact executable artifact:

```bash
repoctl plan -f single-mirror.yaml --artifact > plan.json
```

Review and execute that exact artifact:

```bash
repoctl apply -f single-mirror.yaml --plan-file plan.json
```

Forced actions also require explicit authorization:

```bash
repoctl apply -f single-mirror.yaml --plan-file plan.json --force
```

Dry-run performs the same structural, topology, scope, authorization, and stale-ref preflight without mutation:

```bash
repoctl apply -f single-mirror.yaml --plan-file plan.json --dry-run
```

Imported execution refreshes current repository state but does not rebuild reconciliation intent.

## Version 2 identity boundary

Every source and target ref includes:

- symbolic provider;
- provider-relative repository path;
- runtime Git remote alias;
- branch;
- observed and desired object IDs;
- force intent and planner reason.

Provider/path is durable topology identity. Runtime aliases are execution details. A future multi-mirror executor may map a path-bound target to a current runtime alias after configuration reordering without changing the reviewed target.

URLs, credentials, local filesystem paths, command lines, and array indexes are excluded from identity.

Version 2 rejects missing, absolute, traversal-bearing, transport-like, credential-like, or otherwise unsafe provider paths.

## Version 1 compatibility

Version 1 refs contain provider, runtime alias, and branch but no repository path. They remain parseable for existing single-mirror plan files and tests.

A v1 artifact:

- is matched through the historical single-mirror provider/alias contract;
- cannot authorize multi-mirror targeting;
- must not be reinterpreted as path-bound identity;
- remains covered by the committed v1 schema and golden fixture.

New production observation-to-artifact paths use the strict version-2 constructor. Compatibility-only in-memory callers may continue to create v1 artifacts when provider paths are absent.

## Planning boundary

`repoctl plan` and convenience apply share the same observation-to-plan function. Planning records destructive intent independently of command authorization.

The compatibility `repoctl plan --json` envelope remains `repora.plan` version 1 and remains a view only. Human plan output is also a compatibility view. Use `--artifact` to review exact topology, branches, OID preconditions, and force intent.

An exact artifact is suppressed when selected observation or planning is incomplete.

## Execution boundary

Before mutation, artifact execution validates:

- supported artifact version and kind;
- durable repository UID;
- repository/action cardinality for the current runtime;
- source and target provider/alias ownership;
- version-2 canonical and mirror paths against configuration;
- default-branch scope;
- state/action/force consistency;
- explicit force authorization for real destructive actions;
- every expected source and target OID.

Version-2 path mismatch fails before any repository Git read. Dry-run performs complete stale preflight. Real execution uses the same artifact-backed path.

## Journal compatibility

Execution-record version 2 may reference reconciliation artifact version 1 or 2 through the exact serialized artifact digest. The journal envelope does not change merely because the referenced plan version advances.

Journal action refs remain compatible with the current single-mirror execution model in this slice. Per-target path evidence belongs to the subsequent multi-mirror execution contract.

## Current scope

Artifact v2 establishes stable target binding but the CLI mutation gate remains single-mirror.

The next #15 slice must:

- build multiple path-bound actions;
- resolve each reviewed target to its current runtime alias;
- complete all-target preflight before action zero;
- execute independent actions sequentially and preserve partial outcomes;
- version apply/journal contracts where per-target evidence changes;
- avoid cross-remote atomicity or rollback claims.

The artifact does not model tags, non-default refs, managed files, approvals, provider provisioning, or cross-repository transactions.
