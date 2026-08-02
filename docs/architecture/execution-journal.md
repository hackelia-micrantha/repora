# Execution journal

Repora execution journals provide durable, local-first evidence for planned and attempted repository mutations.

## Current implementation boundary

The implementation defines a versioned internal record model, an in-memory projection from executor results, and an append-only local filesystem writer. Apply and CLI flows do not require or expose journal records yet.

```text
ExecutionRecord {
  version: 1
  kind: repora.io/execution-record
  execution_id: symbolic identifier
  mode: PLAN | DRY_RUN | APPLY
  plan:
    version: plan artifact version
    kind: plan artifact kind
    sha256: digest of the exact deterministic plan serialization
  repository:
    uid: durable repository identity
    id: human-facing repository identity
  actions: ActionRecord[]
}

ActionRecord {
  index: deterministic action order
  type: PUSH_BRANCH
  source: symbolic provider, remote, and branch
  target: symbolic provider, remote, and branch
  before: observed target OID
  desired: planned source OID
  after: resulting OID reported for a successful mutation
  force: bool
  outcome: PLANNED | APPLIED | FAILED | SKIPPED | STALE
  error: optional sanitized diagnostic
}
```

## Executor projection

The journal adapter accepts the exact plan artifact and the executor result produced from that artifact. It fails closed when action counts, indexes, or action values do not match the referenced plan.

Executor evidence maps as follows:

- successful mutations become `APPLIED` and retain the resulting OID;
- stale preflight failures become `STALE`;
- mutation and validation failures become `FAILED`;
- actions not attempted after a failure remain `SKIPPED`;
- unsafe diagnostic strings are replaced with a stable redacted diagnostic.

Partial execution is preserved in plan order. An earlier successful action therefore remains `APPLIED` when a later action fails.

## Local persistence

`journal.Writer` accepts only a validated `Record` and writes one JSON file beneath:

```text
<root>/.repora/journal/<repository-uid>--<execution-id>.json
```

The returned reference is relative to the supplied root and always uses slash separators. Absolute paths are not exposed by the writer.

Persistence properties:

- the caller-owned root must already exist and be a directory;
- `.repora` and `journal` are created one component at a time with mode `0700`;
- existing symlink or non-directory components are rejected before any descendant is created;
- journal files use mode `0600`;
- records are written and synced through a temporary file;
- publication uses an atomic no-replace link, so an existing execution record is never overwritten;
- the journal directory is synced after publication;
- invalid records are rejected before directories are created;
- the resolved journal directory must remain beneath the resolved caller-owned root.

The caller owns selection and lifecycle of the root directory. Retention remains operator-managed.

### Publication outcomes

Before the no-replace link succeeds, an error means no final journal record was published by that call.

After the link succeeds, the final reference is returned even if temporary-file cleanup or directory synchronization fails. The accompanying error explicitly states that publication occurred. Callers must not blindly retry with a new execution identifier; they should inspect the returned reference and treat directory-sync failure as uncertain durability rather than absence.

A collision returns both `ErrRecordExists` and the existing safe reference. The writer never overwrites or invents a replacement identifier.

## Determinism

The plan reference is a SHA-256 digest of the exact validated plan artifact serialization. Repository and action ordering are preserved. The record model deliberately excludes timestamps and generated filesystem paths so identical inputs and execution identifiers serialize identically in tests.

Filesystem uniqueness is provided by the validated repository UID and execution ID.

## Safety and validation

Records fail validation when they contain:

- unsupported versions, kinds, modes, action types, or outcomes;
- invalid repository or execution identifiers;
- malformed plan digests;
- missing or malformed Git object IDs;
- non-sequential action indexes;
- applied actions without a resulting OID;
- URL-like, credential-like, authorization-bearing, or absolute-path-like diagnostic data.

Plan artifacts are validated before journal records are constructed. Journal parsing also rejects unknown JSON fields and trailing data.

The component-by-component path checks prevent an already-present `.repora` or `journal` symlink from redirecting directory creation outside the root. They do not claim a hostile multi-process filesystem sandbox; a stronger adversarial boundary would require platform-specific descriptor-relative operations.

## Deferred integration

Follow-up slices own:

1. fail-closed apply integration when a required journal write fails;
2. persistence of both pre-mutation intent and final execution outcome;
3. independent post-push ref verification where required by policy;
4. CLI output of safe journal references;
5. public JSON schema coordination under issue #3;
6. retention, cleanup, and runtime timestamp policy.

Recovery continues to require re-planning from current repository state. A journal record is evidence of intent and outcome, not authority to replay stale mutations.
