# Execution journal

Status: Current

Repora execution journals provide durable, local-first evidence for dry-run validation and repository mutation.

## Apply integration

CLI `apply` and `sync` operations are journaled by default. The journal root is anchored beside the loaded configuration file:

```text
<config-directory>/.repora/journal/
```

One command-level execution ID is shared across all selected repositories. Each repository writes two immutable entries:

```text
<uid>--<execution-id>--intent.json
<uid>--<execution-id>--result.json
```

Human and current apply output expose safe relative references to the entries.

## Version 3 record model

New path-bound plan artifact v2 executions write `repora.io/execution-record` version 3.

```text
ExecutionRecord {
  version: 3
  kind: repora.io/execution-record
  execution_id: symbolic identifier
  phase: INTENT | RESULT
  mode: PLAN | DRY_RUN | APPLY
  plan:
    version: 2
    kind: repora.io/reconciliation-plan
    sha256: digest of the exact deterministic plan serialization
  repository:
    uid: durable repository identity
    id: human-facing repository identity
  actions: ActionRecord[]
}

ActionRecord {
  index: deterministic action order
  type: PUSH_BRANCH
  source: symbolic provider, provider-relative path, runtime alias, and branch
  target: symbolic provider, provider-relative path, runtime alias, and branch
  before: observed target OID
  desired: planned source OID
  after: resulting OID for an applied mutation
  force: bool
  outcome: PLANNED | VALIDATED | APPLIED | FAILED | SKIPPED | STALE
  error: optional sanitized diagnostic
}
```

Provider/path is durable evidence identity. Runtime aliases are retained only as execution context and cannot retarget a reviewed action.

## Compatibility

- version 1 records remain parseable and retain the historical one-file envelope;
- version 2 intent/result records remain parseable and exclude provider paths;
- version 3 is written for reconciliation artifact v2 and requires safe provider-relative paths;
- version 2 remains written only when consuming a historical reconciliation artifact v1.

All historical schemas remain committed. Consumers must inspect `kind` and `version`.

## Transaction boundary

The audited execution sequence is:

```text
configuration, status, policy, and artifact topology validation
  -> current default-branch validation through runtime bindings
  -> persist INTENT
  -> executor stale-ref preflight for every action
  -> mutate when supported and not dry-run
  -> persist RESULT
  -> render references and return status
```

Required behavior:

- target binding, topology, policy, and branch failures occur before intent persistence;
- intent persistence occurs before executor preflight can reach mutation;
- intent-write failure is fail-closed and performs no mutation;
- dry-run writes a result containing `VALIDATED`, `STALE`, `FAILED`, or `SKIPPED` outcomes;
- real apply writes `APPLIED`, `STALE`, `FAILED`, or `SKIPPED` outcomes;
- runtime failure still attempts final result persistence;
- result-write failure returns nonzero even when a mutation completed;
- an intent record is evidence of reviewed intent, not evidence that mutation occurred;
- a missing result entry means the final outcome must be reconciled against current Git state.

## Multi-mirror dry-run

A multi-mirror dry-run writes one repository-level intent/result pair containing every planned mirror action in deterministic artifact order.

Before intent, Repora verifies:

- durable repository UID;
- canonical provider/path;
- every configured mirror target exactly once;
- complete current status evidence;
- state/action/force agreement under the closed ref policy;
- current canonical and target default branches.

Executor preflight then resolves every expected source and target OID through current runtime aliases. Artifact aliases and array positions are not target authority. If a later target is stale, earlier actions remain `SKIPPED`, the offending action becomes `STALE`, no mutation occurs, and a result record is still attempted.

Real multi-mirror mutation remains gated until continuation and per-target applied-result semantics are complete.

## Plan correlation

Every entry references the exact serialized reconciliation artifact through a SHA-256 digest. Intent and result entries for one repository share:

- execution ID;
- mode;
- plan digest;
- durable repository UID;
- deterministic action ordering, stable provider paths, and before/desired values.

Result projection fails closed when executor actions do not exactly match the referenced plan.

## Local persistence

`journal.Writer` accepts only a validated record and writes beneath the caller-owned existing root.

Persistence properties:

- `.repora` and `journal` are created one component at a time with mode `0700`;
- existing symlink or non-directory components are rejected before descendant creation;
- journal files use mode `0600`;
- content is written and synced through a temporary file;
- publication uses an atomic no-replace link;
- an existing phase entry is never overwritten;
- the journal directory is synced after publication;
- returned references are relative and slash-separated;
- invalid records are rejected before directories are created.

Before the no-replace link succeeds, an error means no final record was published by that call. After publication, the final reference is returned even if temporary-file cleanup or directory synchronization fails; callers treat this as uncertain durability rather than absence.

## Failure evidence

Executor evidence maps as follows:

- successful dry-run preflight becomes `VALIDATED`;
- successful mutation becomes `APPLIED` with the resulting OID;
- stale preflight becomes `STALE`;
- mutation or validation failure becomes `FAILED`;
- unattempted actions remain `SKIPPED`;
- unsafe diagnostics are replaced with a stable redacted message.

## Security boundary

Records exclude:

- runtime transport URLs;
- credentials, tokens, authorization headers, and secret environment values;
- raw command lines;
- absolute cache or checkout paths.

Provider paths must be relative, namespace-qualified, and free of traversal, transport, credential, whitespace, and unsafe delimiter syntax. Validation also rejects URL-like, credential-like, authorization-bearing, and absolute-path-like diagnostics.

## Retention and recovery

Retention is operator-managed. Repora does not automatically delete, compact, upload, sign, or index journal entries.

Safe recovery after a failed or incomplete execution is:

1. inspect the intent and result references returned by the CLI;
2. compare current repository state with the recorded provider paths and before/desired values;
3. rerun status and produce a new exact artifact;
4. review and apply or dry-run the new artifact.

Journal records are evidence, never replay authority. Do not edit expected OIDs or replay an old artifact after current state has changed.

## Deferred capabilities

- retention automation and indexing;
- cryptographic signing or provenance envelopes;
- remote journal storage;
- approval metadata;
- independent post-push provider verification where policy requires more than local remote-tracking evidence.
