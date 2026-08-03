# Reconciliation plan artifact

Status: Current

Repora uses a versioned reconciliation plan artifact as the review and execution boundary for repository mutations.

New plans emit `repora.io/reconciliation-plan` version `2`, described by `schemas/reconciliation-plan-v2.schema.json`. Version 1 remains parseable for historical single-mirror compatibility.

## Operator workflow

Export the exact artifact across one or more mirrors:

```bash
repoctl plan -f repora.yaml --artifact > plan.json
```

Review and validate that exact artifact without mutation:

```bash
repoctl apply -f repora.yaml --plan-file plan.json --dry-run
```

Real apply currently requires a single-mirror configuration:

```bash
repoctl apply -f single-mirror.yaml --plan-file plan.json
repoctl apply -f single-mirror.yaml --plan-file plan.json --force
```

Imported execution refreshes current repository state but does not rebuild or rewrite reconciliation intent.

## Version 2 identity boundary

Every source and target ref includes:

- symbolic provider;
- provider-relative repository path;
- runtime Git remote alias;
- branch;
- observed and desired object IDs;
- force intent and planner reason.

Provider/path is durable topology identity. Runtime aliases are execution details. During multi-mirror dry-run, each reviewed target is matched to current configuration by provider/path and then bound to the current local alias through a separate runtime map. The artifact and its digest remain unchanged.

URLs, credentials, local filesystem paths, command lines, and array indexes are excluded from identity. Version 2 rejects missing, absolute, traversal-bearing, transport-like, credential-like, or otherwise unsafe provider paths.

## Version 1 compatibility

Version 1 refs contain provider, runtime alias, and branch but no repository path.

A v1 artifact:

- remains parseable for historical single-mirror execution;
- is matched through the historical provider/alias contract;
- cannot authorize multi-mirror targeting or preflight;
- must not be reinterpreted as path-bound identity;
- remains covered by the committed v1 schema and golden fixture.

## Planning boundary

Multi-mirror planning matches complete status observations to configured mirrors by provider/path. Actions are emitted in configuration order after identity matching:

- equal mirror: no action;
- behind mirror: normal push intent;
- ahead or diverged mirror: forced overwrite intent.

`repoctl plan --artifact` is the authoritative machine-readable multi-mirror plan. The compatibility `repoctl plan --json` v1 view remains single-mirror only. An exact artifact is suppressed when any selected observation or planning step is incomplete.

## Multi-mirror dry-run boundary

Before intent persistence, imported or convenience artifacts are validated against:

- supported version and kind;
- durable repository UID;
- configured canonical provider/path;
- every configured mirror target exactly once;
- complete current status evidence;
- state/action/force agreement under ref-policy v1;
- current canonical and mirror default branches.

After one repository-level intent record is persisted, the executor resolves every expected source and target OID through the current runtime bindings. All actions are preflighted before action zero and no push occurs.

A missing binding is structural failure, not stale state. An OID mismatch is stale state. A stale later action leaves earlier and later unattempted actions skipped and is preserved in result evidence.

## Real execution boundary

Real multi-mirror mutation remains gated. Current real single-mirror execution additionally requires explicit `--force` authorization for an action already marked forced and uses force-with-lease.

The next mutation slice must reuse the exact same target binding and complete preflight, execute actions sequentially, continue later independent targets after a runtime push failure, and preserve per-target results without rollback claims.

## Journal compatibility

- reconciliation artifact v1 evidence uses execution-record v2;
- reconciliation artifact v2 evidence uses execution-record v3;
- execution-record v3 adds provider path to every source and target ref;
- execution-record v1 and v2 remain parseable and are not reinterpreted.

Intent and result records reference the exact serialized artifact digest.

## Current scope

Artifact v2 and audited dry-run support multiple default-branch mirror actions. The artifact does not model tags, non-default refs, managed files, approvals, provider provisioning, or cross-repository transactions.
