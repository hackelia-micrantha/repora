# Apply JSON version 2

Status: Current

`repoctl apply --json` and `repoctl sync --json` emit `repora.apply` version `2`.

Version 2 adds durable execution-journal correlation to each repository result:

```json
{
  "kind": "repora.apply",
  "version": 2,
  "results": [
    {
      "id": "payments-api",
      "uid": "repo.org.payments-api",
      "state": "BEHIND",
      "applied": false,
      "dry_run": true,
      "actions": [],
      "journal": {
        "execution_id": "run-...",
        "intent": ".repora/journal/repo.org.payments-api--run-...--intent.json",
        "result": ".repora/journal/repo.org.payments-api--run-...--result.json"
      }
    }
  ]
}
```

## Migration from version 1

Consumers must inspect `kind` and `version` before decoding.

Version 2 retains the existing repository and action fields. It adds the optional `journal` object and advances the envelope version because the evidence references are part of the supported contract.

The version 1 schema remains at `schemas/cli-apply-v1.schema.json` for historical consumers. New integrations should validate against `schemas/cli-apply-v2.schema.json`.

## Journal reference semantics

- `execution_id` correlates repositories processed by one command invocation.
- `intent` identifies the immutable pre-execution entry when publication succeeded.
- `result` identifies the immutable final entry when publication succeeded or the writer reported a post-publication durability warning.
- a missing `intent` means required intent publication failed and no mutation was attempted;
- a present intent with a missing result means the final outcome must be reconciled against current Git state;
- references are relative to the directory containing the loaded Repora configuration.

Journal references are evidence locators, not replay handles.

## Failure behavior

JSON output remains a complete document when one repository fails. The repository result includes its safe error and any available journal references, and the process returns nonzero.

A result can report `applied: true` and still return a command failure when final journal persistence fails. Consumers must use both process status and repository result fields.
