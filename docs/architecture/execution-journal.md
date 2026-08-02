# Execution journal

Repora execution journals provide durable, local-first evidence for planned and attempted repository mutations.

## Current implementation boundary

The implementation defines a versioned internal record model and an in-memory projection from executor results. It does not write files, change CLI output, or require a journal record during apply yet.

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

## Determinism

The plan reference is a SHA-256 digest of the exact validated plan artifact serialization. Repository and action ordering are preserved. The record model deliberately excludes timestamps and generated filesystem paths so identical inputs and execution identifiers serialize identically in tests.

A later writer may add filesystem naming and runtime timestamps outside the deterministic evidence payload or through explicitly documented fields.

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

## Deferred integration

Follow-up slices own:

1. a local append-only filesystem writer;
2. fail-closed behavior when a required journal write fails;
3. independent post-push ref verification where required by policy;
4. CLI output of safe journal references;
5. public JSON schema coordination under issue #3;
6. retention, cleanup, and runtime timestamp policy.

Recovery continues to require re-planning from current repository state. A journal record is evidence of intent and outcome, not authority to replay stale mutations.
