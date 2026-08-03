# ADR-0012: Closed reference synchronization policy v1

Status: Implemented

Decision date: 2026-08-02

Last reviewed: 2026-08-02

Supersedes: none

Superseded by: none

Implemented by: PR #75

Related issues: #2, #4, #13, #15

## Context

Repora already reconciles only one canonical default branch to one mirror default branch. Ahead and diverged mirrors require explicit force authorization and use force-with-lease. That boundary existed partly as code assumptions and command behavior rather than one explicit policy contract.

Adding multi-mirror support while leaving reference scope implicit would make later branch, tag, or wildcard expansion easy to introduce accidentally or inconsistently.

## Decision

Adopt a closed version-1 reference policy with exactly these effective values:

- scope: `default-branch-only`;
- destructive behavior: `require-force`.

The policy may be written per repository or omitted. Omission normalizes to the same values.

Configuration loading rejects unsupported versions or values. Planning maps `EQUAL`, `BEHIND`, `AHEAD`, and `DIVERGED` to no-op, normal push, or forced overwrite intent. Mutation authorization remains separate from planning.

Version 1 does not support tags, non-default branches, wildcard refspecs, deleted-ref reconciliation, or provider API protection enforcement.

## Alternatives

### Keep behavior implicit

Rejected because future topology work could duplicate or weaken the mutation boundary.

### Add a general branch/tag allowlist now

Rejected because no implemented runtime supports those operations and the larger schema would imply capabilities Repora does not have.

### Treat `--force` as the policy model

Rejected because a command authorization flag does not define eligible refs or artifact interpretation.

## Consequences

- Existing configurations remain compatible.
- Unsupported expansion fails early instead of being ignored.
- Planner behavior is driven by an explicit versioned policy.
- The policy model remains intentionally small and cannot express future branch/tag use cases.
- A future policy version with alternate accepted behavior must coordinate artifact compatibility and migration.

## Security implications

- Only current default branches are eligible.
- Destructive intent is visible in the reviewed plan.
- Real destructive mutation requires explicit `--force`.
- Stale-ref validation and force-with-lease remain independent safeguards.

## Validation

- omitted and explicit supported policy tests;
- unsupported version, scope, and destructive-mode rejection tests;
- relationship decision matrix tests;
- existing planner force and determinism tests.
