# Apply output v3 migration

Status: Current

`repora.apply` version 3 is the per-target result contract for path-bound multi-mirror apply and dry-run.

## Why the version changed

Apply v2 reports planned action arguments but does not expose an explicit outcome, observed target OID, desired source OID, or resulting OID for every mirror. Those fields are required to represent non-atomic multi-mirror execution honestly.

Version 3 adds one complete action result for every reviewed target action.

## Envelope

```json
{
  "kind": "repora.apply",
  "version": 3,
  "results": []
}
```

Each repository result contains:

- `id` and durable `uid`;
- aggregate repository `state`;
- `applied`, true only when a real execution completed all required actions or was a successful no-op;
- `dry_run`;
- deterministic `actions` in exact artifact order;
- optional journal references;
- optional aggregate error.

## Action result

Each action includes:

- `type`;
- stable provider/path/branch `source` and `target` strings;
- `force` intent;
- `before` target OID;
- `desired` source OID;
- optional `after` OID;
- `outcome`;
- optional sanitized error.

Outcomes are:

- `VALIDATED` — dry-run preflight succeeded;
- `APPLIED` — real mutation succeeded and `after` is present;
- `FAILED` — structural or runtime action failure;
- `STALE` — expected source or target OID changed before action zero;
- `SKIPPED` — action was not attempted because complete preflight failed;
- `PLANNED` — reserved for pre-execution projection and not expected in a successful final execution response.

## Partial success

Multi-mirror mutation is not atomic. A result may contain:

```text
APPLIED, FAILED, APPLIED
```

Later independent actions are attempted after a runtime failure. Earlier successful actions are not rolled back. The command returns exit `1` when any repository or target fails while preserving all available action outcomes.

## Force behavior

A real execution containing any action already marked `force: true` requires the command-level `--force` authorization before journal creation or mutation. The flag does not convert normal actions into forced actions.

Dry-run validates forced intent without requiring authorization.

## Compatibility

- single-mirror-only apply commands continue to emit apply v2;
- any selected configuration containing a multi-mirror repository uses apply v3 for every selected repository;
- exact plan artifact v2 is required by the path-bound execution route;
- apply v2 and its schema remain committed and supported;
- consumers must inspect both `kind` and `version`.

## Retry

After any stale or runtime failure:

1. inspect apply v3 and execution-record v3 evidence;
2. run status across all mirrors;
3. produce a new exact artifact;
4. review force intent again;
5. execute the new artifact.

Do not replay journal evidence or edit expected OIDs.
