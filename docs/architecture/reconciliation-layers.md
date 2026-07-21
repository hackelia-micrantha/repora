# Reconciliation layer ownership

Repora's reconciliation dependency direction is:

```text
config/topology
    -> transport resolver
    -> observed repository state
    -> planner
    -> executor
```

## Ownership

- `internal/config` owns durable repository identity and configured provider-relative locations.
- `internal/transport` owns runtime transport resolution. Resolved URLs are runtime state and are not repository identity.
- `internal/status` owns observation of fetched references and their `EQUAL`, `BEHIND`, `AHEAD`, or `DIVERGED` relationship.
- `internal/plan` owns deterministic reconciliation decisions. Given identical topology, observations, and force policy, it returns an identical in-memory reconciliation plan. Planning performs no Git mutations.
- `internal/executor` owns Git mutation side effects. It executes planner-produced actions in order, validates complete plans before mutation, and does not inspect repository status or re-decide reconciliation policy.
- `internal/apply` orchestrates execution preparation: it resolves current branches and lease observations, invokes the planner, preserves the existing output model, and delegates mutations to the executor.

## Current slice

The planner decides whether reconciliation produces no action, a normal branch push, a fail-closed forced action, or an explicitly authorized forced action. Dry-run renders those planner actions directly. Real apply passes the same in-memory actions to the executor, which owns `PushBranch` and `ForcePushBranchWithLease` calls.

The executor's action results are internal implementation types and are not a stabilized public serialization contract. The existing user-facing `repoctl plan` JSON model remains separate from the in-memory reconciliation model, avoiding premature coupling to issue #8 or the public JSON contracts owned by issue #3.

## Temporary boundary

Apply still resolves default branches and the expected mirror head used by force-with-lease before invoking the planner. Those reads are observation and execution preparation, not mutation. The command path also continues to expose the existing apply result model rather than a generalized execution-result contract.

## Remaining issue #22 slices

1. Add structured per-repository and per-mirror partial-failure results.
2. Strengthen proof that dry-run and apply consume exactly the same complete action sequence as multi-action planning evolves.
3. Add stale-plan validation before execution.
4. Integrate the serialized plan artifact under issue #8 without coupling durable identity to runtime URLs.
5. Extend planner and executor behavior to multiple mirrors under issues #13 and #15.
