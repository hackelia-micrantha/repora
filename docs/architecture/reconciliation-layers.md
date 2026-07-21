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
- `internal/apply` orchestrates execution preparation: it resolves current branches and lease observations, invokes the planner once, projects compatibility output from that exact plan, and delegates the same plan to the executor.

## Current slice

Apply now has one observation-to-plan path. Dry-run renders compatibility actions projected from the resulting in-memory plan. Real apply renders the same projected actions and passes that same plan instance to the executor without rebuilding or reinterpreting reconciliation decisions.

Tests cover safe pushes, forced pushes with lease observations, and no-action plans to prove dry-run and apply report identical action fields. Forced execution tests also prove the reported source, target, force flag, and expected lease match the mutation arguments supplied to Git.

Executor results retain one ordered internal result for every planned action. Successful actions before a mutation failure remain `APPLIED`, the failing action is `FAILED` with its error, and unattempted later actions remain `SKIPPED`. Complete-plan validation still occurs before any mutation. These internal types are not a stabilized public serialization contract.

## Failure and recovery

Planner failures produce no mutation attempts. Mutation execution stops after the first failure. Callers must preserve the returned partial result for diagnostics and must re-plan from current remote state before retrying rather than replaying the prior in-memory plan. The existing user-facing apply JSON remains unchanged and does not claim the repository was fully applied after a partial failure.

Apply still resolves default branches and the expected mirror head used by force-with-lease before invoking the planner. Those reads are observation and execution preparation, not mutation. The command path continues to expose the existing apply result model rather than a generalized execution-result contract.

## Remaining issue #22 slices

1. Add stale-plan validation before execution.
2. Integrate the serialized plan artifact under issue #8 without coupling durable identity to runtime URLs.
3. Add public per-repository and per-mirror partial-failure reporting after the multi-mirror result model is defined.
4. Extend planner and executor behavior to multiple mirrors under issues #13 and #15.
