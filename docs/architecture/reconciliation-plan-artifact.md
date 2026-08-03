# Reconciliation plan artifact

Status: Current

Repora uses a versioned reconciliation plan artifact as the review and execution boundary for repository mutations.

New plans emit `repora.io/reconciliation-plan` version 2. Version 1 remains parseable for historical single-mirror compatibility.

## Operator workflow

```bash
repoctl plan -f repora.yaml --artifact > plan.json
repoctl apply -f repora.yaml --plan-file plan.json --dry-run --json
repoctl apply -f repora.yaml --plan-file plan.json --json
```

When any reviewed action is forced:

```bash
repoctl apply -f repora.yaml --plan-file plan.json --force --json
```

Imported execution refreshes current repository observations and bindings but never rebuilds, edits, reorders, or retargets the artifact.

## Version 2 identity boundary

Every source and target ref contains:

- provider;
- provider-relative repository path;
- serialized runtime Git alias;
- branch;
- observed and desired OIDs;
- force intent and planner reason.

Provider/path is durable identity. The serialized alias is context only. At execution time each reviewed target binds to the current local alias through a separate runtime map; the artifact and its digest remain unchanged.

URLs, credentials, local paths, command lines, and array indexes are excluded. Provider paths reject absolute, traversal-bearing, transport-like, credential-like, whitespace, and unsafe delimiter syntax.

## Version 1 compatibility

Version 1 refs have no provider path. They remain supported only through the historical single-mirror provider/alias execution path and cannot authorize multi-mirror targeting.

## Planning

Complete status observations are matched to configuration by provider/path. Actions are emitted in configuration order:

- `EQUAL`: no action;
- `BEHIND`: normal push intent;
- `AHEAD` or `DIVERGED`: forced overwrite intent.

`plan --artifact` is the authoritative multi-mirror machine contract. The compatibility `plan --json` v1 view remains single-mirror only. Incomplete selected planning suppresses the artifact.

## Selected preparation

Before any selected repository mutates, Repora:

- observes every selected repository;
- builds or splits one exact artifact per repository;
- validates artifact v2, kind, cardinality, and serialization;
- refuses the complete selected execution if any preparation fails;
- checks whether any action requires command-level force authorization.

Preparation or missing authorization writes no intent and performs no mutation.

## Repository execution

Before intent persistence, each artifact is checked against current UID, canonical provider/path, every configured mirror, complete status, state/action/force agreement, and current default branches.

After immutable intent persistence, every expected source and target OID is validated before action zero.

If preflight succeeds:

- actions execute sequentially in artifact order;
- normal intent uses normal push;
- forced intent uses force-with-lease and the reviewed old target OID;
- runtime failure marks that action failed and later independent actions continue;
- successful actions are not rolled back;
- apply v3 and execution-record v3 preserve all outcomes;
- any failure returns nonzero and retry requires a new artifact.

The same exact-artifact path serves convenience apply and `--plan-file` execution.

## Public result and evidence contracts

- single-mirror-only command selections retain apply v2;
- mixed or multi-mirror selections emit apply v3;
- artifact v2 execution writes execution-record v3;
- artifact v1 can still write record v2 through the legacy path;
- historical schemas remain committed.

Apply v3 identifies source and target by stable provider/path/branch and includes force, before, desired, optional after, outcome, and sanitized error.

## Current scope

Artifact v2 models zero or one default-branch action per configured mirror. It does not model tags, non-default refs, managed files, approvals, provider provisioning, rollback, or cross-repository transactions.
