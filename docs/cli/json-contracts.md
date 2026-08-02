# CLI JSON contracts

Repora's machine-readable CLI output is versioned independently from the binary.

## Contract envelope

A stabilized output includes:

- `kind`: identifies the command-specific document type;
- `version`: identifies the schema version for that kind;
- repository `uid`: durable identity for correlation across renames or topology moves;
- repository `id`: the current human-facing alias.

The stabilized response contracts are:

| Command | Kind | Version | Schema |
| --- | --- | ---: | --- |
| `repoctl status --json` | `repora.status` | 1 | `schemas/cli-status-v1.schema.json` |
| `repoctl plan --json` | `repora.plan` | 1 | `schemas/cli-plan-v1.schema.json` |
| `repoctl apply --json` and `repoctl sync --json` | `repora.apply` | 1 | `schemas/cli-apply-v1.schema.json` |

The exact executable plan contract is separate:

| Command | Kind | Version | Schema |
| --- | --- | ---: | --- |
| `repoctl plan --artifact` | `repora.io/reconciliation-plan` | 1 | `schemas/reconciliation-plan-v1.schema.json` |

`repoctl apply --plan-file` accepts that exact reconciliation artifact. It does not accept the `repora.plan` compatibility response as executor input.

## Compatibility rules

Within one `kind` and `version`:

- existing field names and meanings do not change;
- required fields are not removed or made optional;
- enum values are not renamed or repurposed;
- consumers must not use provider/path or resolved URLs as durable identity;
- output ordering remains deterministic for identical ordered inputs.

A breaking change requires a new integer `version` and a new schema file. Additive fields may remain within a version only when they are optional and do not change existing interpretation. Repora currently uses strict schemas for the stabilized v1 documents, so additions should normally be introduced through a new version until an explicit extension policy is adopted.

Human-readable output is not covered by these JSON schemas.

## Plan response versus executable artifact

`repora.plan` version 1 remains supported for automation compatibility. It is a simplified response containing mirror target, commit distance, and destructive classification.

It is now projected from the exact reconciliation plans used to create `repora.io/reconciliation-plan`. It does not make independent planning decisions.

Consumers that need exact branches, source and target OID preconditions, force intent, reason, artifact validation, or later execution must use `plan --artifact`.

## Validation

Representative complete documents are checked against committed golden fixtures. Contract changes therefore require an explicit schema, fixture, test, and compatibility review rather than silently following internal Go struct changes.

Downstream consumers should inspect `kind` and `version` before decoding the command-specific payload and validate documents against the corresponding schema when strict compatibility matters.

## Scope boundaries

CLI response contracts describe command results for automation and tests. The reconciliation artifact is durable review and execution input. Execution journals and evidence records use separate kinds and versions under issues #6 and #1.
