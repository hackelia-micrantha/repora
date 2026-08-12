# Managed artifact plan v1

Status: Current

## Scope

This document describes the implemented serialized contract and parser/validator for `repora.io/managed-artifact-plan` version 1.

It does **not** mean canonical Git observation, README plan generation, dry-run, commit creation, or remote apply is implemented. Those runtime steps remain proposed in [`managed-artifacts.md`](managed-artifacts.md) and are owned by issue #12.

## Purpose

The managed artifact plan is the exact review boundary for future README changes. It remains separate from `repora.io/reconciliation-plan`, which continues to model Git-reference reconciliation only.

Schema: [`../../schemas/managed-artifact-plan-v1.schema.json`](../../schemas/managed-artifact-plan-v1.schema.json)

Example: [`../../examples/managed-artifact-plan-v1.json`](../../examples/managed-artifact-plan-v1.json)

## Top-level contract

```text
Plan {
  kind: "repora.io/managed-artifact-plan"
  version: 1
  repositories: RepositoryPlan[]
}
```

`repositories` must be explicitly present. An empty array is a valid no-change plan. If a repository entry is present, v1 requires exactly one README action.

Repository order is serialized order. A future planner is responsible for emitting a deterministic order from the same inputs.

## Repository identity

Each repository plan contains:

- durable `uid`;
- human-facing `id`;
- canonical `provider`, provider-relative `path`, and default `branch`;
- exact reviewed canonical `base_oid`;
- one `WRITE_README` action.

UIDs, IDs, and canonical provider/path/branch identity must be canonical strings without surrounding whitespace. Duplicate UIDs, IDs, or canonical targets fail validation.

The plan intentionally stores provider/path identity instead of a resolved transport URL so credentials, host-specific URL formatting, and runtime transport policy do not become durable plan identity.

## README action

V1 permits only:

```text
type: WRITE_README
path: README.md
```

No other managed output path or action type is valid.

### Observed state

When `README.md` exists:

```json
{
  "present": true,
  "mode": "100644",
  "sha256": "<lowercase SHA-256>"
}
```

Allowed regular-file modes are `100644` and `100755`.

When it does not exist:

```json
{
  "present": false
}
```

Mode and digest must be absent in the missing-file representation. This makes absence distinct from an empty file.

### Desired state

Desired state contains:

- reviewed Git mode;
- lowercase SHA-256 digest;
- exact UTF-8 README content.

The parser recomputes SHA-256 from the exact serialized content and rejects a mismatch. Existing regular README mode must be preserved; a newly created README must use `100644`.

Desired README text:

- is limited to 1 MiB;
- must use LF line endings;
- must be valid UTF-8;
- may contain tab and newline controls;
- rejects terminal/control characters such as ANSI escape.

An existing README whose observed digest/mode already equals desired state must not appear as a no-op action.

## Template evidence

`template_sha256` is a lowercase SHA-256 digest of the template bytes used for planning.

The durable plan does not contain the template filesystem path. That path is runtime/configuration input rather than portable review identity.

## Review diff field

Every action includes a non-empty `diff` string with fixed labels:

```text
--- a/README.md
+++ b/README.md
@@ ...
```

The parser requires:

- valid UTF-8;
- LF line endings/control-safe text;
- fixed README labels followed immediately by a unified hunk;
- a bounded serialized size;
- a final LF.

This contract validates the review-diff envelope. Exact diff generation and verification against the observed Git blob belong to the planner/apply slices, because the plan deliberately stores only the observed content digest rather than duplicating old README content.

## Strict parsing

The Go parser:

- rejects unknown JSON fields;
- rejects `null` anywhere in v1;
- rejects trailing JSON values/data;
- preserves required false/empty values through pointer-backed fields where absence must remain distinguishable;
- validates duplicate repository identity/target constraints;
- validates cross-field mode/content/digest invariants that JSON Schema alone cannot express.

Marshal validates first and uses struct-only JSON serialization, so identical in-memory plan values serialize deterministically.

## Security boundary

The contract contains no:

- credential-bearing transport URL;
- template or cache filesystem path;
- environment value;
- Git command line;
- timestamp;
- author/committer identity;
- generic filesystem output path;
- executable template instructions.

Unsafe control characters are rejected from desired README text and review diff content so future human-readable plan rendering cannot be altered by terminal escape sequences.

## Deferred runtime work

Issue #12 still owns:

1. configuration-root template loading with symlink containment;
2. canonical README Git-tree observation and mode validation;
3. deterministic byte-faithful diff generation;
4. managed plan construction and human review output;
5. exact stale preflight and dry-run;
6. isolated commit creation and guarded canonical push;
7. result/evidence rendering;
8. end-to-end proof that unconfigured repositories remain untouched.
