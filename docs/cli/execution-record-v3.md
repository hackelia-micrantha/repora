# Execution record v3 migration

Status: Current

Execution record v3 is the path-bound intent/result evidence contract for reconciliation artifact v2.

## Why the version changed

Execution record v2 identifies refs by provider, runtime Git alias, and branch. Runtime aliases may change when mirrors are reordered and therefore cannot provide durable target evidence for multi-mirror execution.

Version 3 adds provider-relative `path` to every source and target ref.

## Consumer changes

Consumers must inspect `kind` and `version` before decoding.

For version 3:

```json
{
  "version": 3,
  "kind": "repora.io/execution-record",
  "actions": [
    {
      "source": {
        "provider": "gitlab",
        "path": "org/repository",
        "remote": "canonical",
        "branch": "main"
      },
      "target": {
        "provider": "github",
        "path": "org/repository",
        "remote": "mirror-0",
        "branch": "main"
      }
    }
  ]
}
```

Use `provider + path` as durable target evidence. Treat `remote` as historical runtime context only.

## Compatibility

- version 1 remains parseable as the historical one-file record;
- version 2 remains parseable as the historical intent/result record without paths;
- version 3 is emitted for reconciliation artifact v2;
- version 2 may still be emitted when consuming a historical reconciliation artifact v1;
- old records are never upgraded or reinterpreted in place.

## Outcome semantics

Intent actions use `PLANNED`.

Dry-run result actions use:

- `VALIDATED` when expected refs match;
- `STALE` for the action whose expected ref changed;
- `FAILED` for validation or operational failure attributable to an action;
- `SKIPPED` for actions not reached after failure.

Real multi-mirror applied-result semantics are not yet public. The real mutation gate remains closed until the per-target apply contract is versioned.

## Security

Provider paths must be relative, namespace-qualified, and free of schemes, hosts, credentials, traversal, whitespace, backslashes, and unsafe delimiters. URLs and absolute local paths remain excluded from journal evidence.
