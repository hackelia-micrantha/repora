# Execution journal

Status: Current

Repora execution journals provide durable local evidence for dry-run validation and real repository mutation.

## Storage and correlation

The journal root is beside the loaded configuration:

```text
<config-directory>/.repora/journal/
```

One command execution ID is shared across selected repositories. Each repository writes an immutable pair:

```text
<uid>--<execution-id>--intent.json
<uid>--<execution-id>--result.json
```

Every record references the exact serialized reconciliation artifact by SHA-256 digest. Intent and result preserve the same UID, mode, plan digest, action order, provider paths, and before/desired values.

## Execution record v3

Path-bound artifact v2 execution writes `repora.io/execution-record` version 3.

Each action contains:

- deterministic index and `PUSH_BRANCH` type;
- source and target provider, provider-relative path, runtime alias, and branch;
- observed target OID (`before`);
- planned source OID (`desired`);
- resulting OID (`after`) for successful real mutation;
- force intent;
- `PLANNED`, `VALIDATED`, `APPLIED`, `FAILED`, `SKIPPED`, or `STALE` outcome;
- optional sanitized error.

Provider/path is durable evidence identity. Runtime aliases are historical execution context only.

## Compatibility

- version 1 remains the historical one-file record;
- version 2 remains the historical intent/result record without provider paths;
- version 3 requires artifact v2 and safe provider paths;
- artifact v1 can still produce record v2 through the legacy single-mirror path;
- old records are never upgraded or reinterpreted.

## Audited execution sequence

```text
prepare all selected repositories and exact artifacts
  -> authorize any reviewed forced actions
  -> for each repository:
       validate topology, state/action/force intent, and default branches
       persist INTENT
       preflight every expected source and target OID
       dry-run or execute actions
       persist RESULT
  -> expose references and return aggregate status
```

Required behavior:

- preparation or authorization failure writes no intent and performs no selected mutation;
- topology, policy, or branch failure occurs before intent;
- intent-write failure prevents every push in that repository;
- stale preflight writes ordered `SKIPPED`/`STALE` result evidence and performs no push;
- real execution records every attempted target independently;
- runtime failure does not prevent later independent actions;
- result-write failure remains nonzero even if pushes completed;
- an intent record is not evidence that mutation occurred;
- a missing result requires reconciliation against current Git state.

## Partial real execution

After complete repository preflight, mirrors execute sequentially in artifact order. A result may contain:

```text
APPLIED, FAILED, APPLIED
```

The result record preserves:

- successful targets with `after = desired`;
- failed targets with sanitized diagnostics;
- later successful targets attempted after failure;
- no automatic rollback or inverse push.

This is intentional non-atomic evidence. Retry requires a new status observation and exact artifact.

## Local persistence

`journal.Writer` writes only validated records beneath the caller-owned root.

- `.repora` and `journal` directories use mode `0700`;
- existing symlink or non-directory components are rejected;
- files use mode `0600`;
- temporary content is synced before atomic no-replace publication;
- an existing phase record is never overwritten;
- the journal directory is synced after publication;
- returned references are safe relative slash-separated paths;
- invalid records are rejected before directories are created.

After publication, a cleanup or directory-sync error still returns the published reference and a nonzero error so callers treat durability as uncertain rather than absent.

## Security boundary

Records exclude transport URLs, credentials, tokens, authorization headers, command lines, and absolute local paths. Provider paths must be relative, namespace-qualified, and free of traversal, transport, credential, whitespace, and unsafe delimiter syntax.

Unsafe diagnostics are replaced by a stable redacted message before public or journal serialization.

## Recovery

After stale, partial, or uncertain execution:

1. inspect apply v3 and journal references;
2. compare current mirrors with recorded provider paths and OIDs;
3. rerun status for every target;
4. create a new exact artifact;
5. review force intent and execute the new artifact.

Journal records are evidence, never replay authority. Do not edit OIDs, replay an old record, or construct an unreviewed rollback.

## Deferred capabilities

- retention automation and indexing;
- signing or provenance envelopes;
- remote journal storage;
- approval metadata;
- provider-side post-push verification beyond current Git evidence.
