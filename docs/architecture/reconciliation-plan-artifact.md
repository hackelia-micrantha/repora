# Reconciliation plan artifact

Status: Current

Repora uses a versioned reconciliation plan artifact as the review and execution boundary for repository mutations.

The current artifact is `repora.io/reconciliation-plan` version `1` and is described by `schemas/reconciliation-plan-v1.schema.json`.

## Operator workflow

Export the exact executable artifact:

```bash
repoctl plan -f repora.yaml --artifact > plan.json
```

Review and validate `plan.json`, then execute that exact artifact:

```bash
repoctl apply -f repora.yaml --plan-file plan.json
```

When the artifact contains any forced action, execution also requires explicit authorization:

```bash
repoctl apply -f repora.yaml --plan-file plan.json --force
```

`apply --plan-file` refreshes the selected repositories from configuration so executor stale-ref checks compare the artifact against current remote-tracking refs. It does not rebuild reconciliation intent.

A plan file may contain a configuration-ordered subset of repositories. Repository selection uses durable `uid`; a missing or duplicate UID fails before repository observation or mutation.

## Contents

Each artifact contains:

- repository `uid` for durable identity and `id` for display;
- one or more semantic ref-update actions;
- symbolic source and target provider, remote, and branch values;
- the observed target object ID and desired source object ID;
- the force flag and planner reason.

Transport URLs, credentials, command lines, and local filesystem paths are excluded.

## Planning boundary

`repoctl plan` builds the artifact through the same observation-to-plan function used by convenience apply.

The stabilized `repoctl plan --json` response remains `repora.plan` version `1` for compatibility. It is now a projection from the exact reconciliation plans represented by the artifact; it does not make independent mutation decisions.

Human plan output remains a compatibility view. Use `--artifact` when branches, force flags, object-ID preconditions, and exact executor input must be reviewed.

An exact artifact is not emitted when any selected repository cannot be observed or planned completely. This avoids presenting a partial document as a complete executable plan.

## Execution boundary

Convenience apply builds an artifact and delegates to the same artifact execution function used by `--plan-file`.

Before mutation, artifact execution validates:

- artifact version, kind, repositories, actions, refs, OIDs, and serialized safety constraints;
- durable repository UID against configuration;
- canonical and mirror provider/remote ownership;
- explicit `--force` authorization for forced actions;
- current source and target OIDs for every action.

The executor rejects the complete repository plan before action zero when any structural or stale-ref check fails.

The artifact is review evidence, not a promise that execution remains safe indefinitely. Operators should re-plan from current state instead of editing or replaying stale artifacts.

## Compatibility and identity

Repository matching uses `uid`, not the human-facing `id`, provider/path location, or resolved URL. A harmless alias change therefore does not redefine durable repository identity.

The v1 `repora.plan` CLI schema remains supported as a compatibility response. The reconciliation artifact schema is the authoritative executable contract.

## Current scope

Version 1 models default-branch Git ref reconciliation only. It does not model managed file diffs, workflow diffs, multi-mirror targeting, approvals, or cross-repository transactions.

Future domains may reuse versioned envelope and safety conventions, but they require domain-specific action schemas unless implemented experience demonstrates a genuinely shared abstraction.
