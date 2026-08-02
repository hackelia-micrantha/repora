# Reconciliation plan artifact

Repora uses a versioned reconciliation plan artifact as the boundary between planning and execution.

The current artifact is `repora.io/reconciliation-plan` version `1` and is described by `schemas/reconciliation-plan-v1.schema.json`.

## Contents

Each artifact contains:

- repository `uid` for durable identity and `id` for display;
- one or more semantic ref-update actions;
- symbolic source and target provider, remote, and branch values;
- the observed target object ID and desired source object ID;
- the force flag and planner reason.

Transport URLs, credentials, command lines, and local filesystem paths are excluded.

## Execution boundary

The planner creates the artifact from observed repository state. The executor validates the artifact, rechecks the expected source and target object IDs, and fails closed when the plan is stale or malformed before attempting a push.

The artifact is review evidence, not a promise that execution remains safe indefinitely. Operators should re-plan from current state instead of replaying stale artifacts.

## Current scope

Version 1 models default-branch Git ref reconciliation only. It does not yet model managed file diffs, multi-mirror targeting, journals, approvals, or cross-repository transactions. Those additions require explicit follow-up work and compatible artifact evolution.
