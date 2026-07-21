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
- `internal/apply` temporarily owns both the mechanical preparation needed by the current Git wrapper and mutation side effects.

## Current slice

The planner now decides whether reconciliation produces no action, a normal branch push, a fail-closed forced action, or an explicitly authorized forced action. Apply converts planner actions into the existing human-readable and JSON result model and executes them without re-deciding repository-state policy.

The existing user-facing `repoctl plan` JSON model remains separate from the new in-memory reconciliation model. This avoids prematurely stabilizing the serialized plan artifact owned by issue #8 or the public JSON contracts owned by issue #3.

## Temporary boundary

Apply still resolves default branches and the expected mirror head used by force-with-lease before invoking the planner. Those reads are execution preparation, not mutation. A later issue #22 slice should introduce an executor that accepts an already-complete plan and owns side effects only.

## Remaining issue #22 slices

1. Make real apply consume planner output exclusively across the full command path.
2. Extract an executor that owns side effects only.
3. Add structured per-repository and per-mirror partial-failure results.
4. Prove dry-run and apply consume exactly the same action sequence.
5. Add stale-plan validation before execution.
6. Integrate the serialized plan artifact under issue #8 without coupling durable identity to runtime URLs.
