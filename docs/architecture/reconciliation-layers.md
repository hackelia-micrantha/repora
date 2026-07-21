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
- `internal/executor` owns Git mutation side effects and stale-input enforcement. It validates complete plans, verifies expected refs, and then executes planner-produced actions without re-deciding reconciliation policy.
- `internal/apply` orchestrates execution preparation: it resolves current branches and ref observations, invokes the planner once, projects compatibility output from that exact plan, and delegates the same plan to the executor.

## Current slice

Apply has one observation-to-plan path. For every action-producing state, it captures the canonical source OID and mirror target OID along with branch names. Dry-run renders compatibility actions projected from that plan. Real apply passes the same plan to the executor.

Immediately before any mutation, the executor resolves every planned source and target ref again. If any ref differs from its planned OID, cannot be resolved, or the plan omits an expected OID, the complete execution fails closed without performing any mutation. Structural and stale-ref preflight checks cover the complete action sequence before action zero executes.

The existing user-facing apply JSON remains unchanged. Expected source OIDs and non-forced target guards remain internal plan data; forced actions continue to expose only the existing force-with-lease target field.

Executor results retain one ordered internal result for every planned action. Successful actions before a mutation failure remain `APPLIED`, the failing action is `FAILED` with its error, and unattempted later actions remain `SKIPPED`. Preflight failures mark the offending action failed while all other actions remain skipped.

## Failure and recovery

Planner, structural-validation, and stale-plan failures produce no mutation attempts. Mutation execution stops after the first runtime failure. Callers must preserve returned partial results for diagnostics and must re-plan from current remote state before retrying rather than replaying prior in-memory decisions.

Force-with-lease remains a final remote-side target guard, but it does not replace executor preflight: stale validation also protects normal pushes, detects source drift, and validates the complete action sequence before mutation begins.

## Remaining issue #22 coordination

1. Integrate the serialized plan artifact under issue #8 without coupling durable identity to runtime URLs.
2. Add public per-repository and per-mirror partial-failure reporting after the multi-mirror result model is defined.
3. Extend planner and executor behavior to multiple mirrors under issues #13 and #15.
