# Reconciliation plan artifact v2

New `repoctl plan --artifact` output uses:

```json
{
  "kind": "repora.io/reconciliation-plan",
  "version": 2,
  "repositories": []
}
```

Version 2 adds a provider-relative `path` to every source and target ref.

## Why the version changed

Version 1 identified refs by provider, runtime remote alias, and branch. That cannot distinguish several mirrors of the same provider or survive configuration reordering safely.

Version 2 binds the reviewed action to declarative provider/path topology. Runtime aliases remain present for execution compatibility but are not target identity.

## Migration

Consumers should:

1. validate `kind` and `version`;
2. use `source.provider + source.path` and `target.provider + target.path` as topology identity;
3. treat `remote` as an execution alias only;
4. reject missing or unsafe paths in v2;
5. never infer a target from action or mirror array position.

## Version 1

Version 1 remains parseable for existing single-mirror plan files. It has no `path` field and remains matched through the historical provider/alias contract.

Version 1 cannot authorize future multi-mirror execution and is not upgraded or reinterpreted automatically.

Historical schema: `schemas/reconciliation-plan-v1.schema.json`.

Current schema: `schemas/reconciliation-plan-v2.schema.json`.

## Current runtime boundary

Artifact v2 is used for new single-mirror plan/apply flows. Multi-mirror plan/apply/sync remain gated until per-target planning, complete preflight, independent outcomes, and journal evidence are implemented.
