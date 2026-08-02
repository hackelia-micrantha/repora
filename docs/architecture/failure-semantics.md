# Failure and recovery semantics

Status: Current

Repora controls durable Git state. Failures must therefore be explicit in human output, machine-readable output, and process exit status.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | The requested operation completed successfully. For dry-run, planning completed without mutation. |
| `1` | Configuration, observation, planning, execution, serialization, or other operational failure. |
| `2` | Repository state requires explicit destructive authorization, such as an ahead or diverged mirror without an accepted force path. |

A command must not return `0` merely because it successfully serialized or printed a failed result.

## Processing stages

```text
load configuration
  -> observe repository state
  -> classify relationship
  -> build plan
  -> validate executable artifact
  -> stale-ref preflight
  -> mutate
  -> aggregate results
  -> render output
  -> return exit status
```

Each stage has a distinct failure boundary.

## Configuration failures

Configuration is parsed with strict field checking and validated before repository work begins.

A configuration failure:

- performs no Git operations;
- writes a diagnostic to stderr;
- emits no normal command result;
- returns exit code `1`.

Credential-bearing HTTP URLs, unsupported providers, unsupported topology cardinality, and invalid provider-relative paths fail during this boundary.

## Observation failures

Observation includes cache preparation, remote configuration, fetch, remote HEAD setup, and divergence classification.

Without `--continue-on-error`, observation still completes through the bounded worker set, but normal command output is suppressed after any repository observation fails and the command returns `1`.

With `--continue-on-error`:

- independently observed repositories remain available for output;
- failed repository observations are omitted from normal result collections;
- status and plan return non-zero;
- apply refuses to proceed unless the existing force path explicitly permits processing only successfully observed repositories.

Observation failure and destructive-state refusal are different conditions. Current status/plan aggregation may return `2` when both occur because the unsafe-state code has precedence. This should be revisited when the CLI error taxonomy is stabilized.

## Planning failures

Planning must be pure with respect to remotes.

A planning failure:

- performs no mutation;
- returns the repository identity and available compatibility result context where possible;
- returns exit code `1` from apply;
- requires correction or a fresh observation before retry.

Planning must fail closed for unsupported state, ambiguous topology, missing branches, missing OIDs, or force-required state without authorization.

## Artifact validation failures

The executor accepts only a validated versioned reconciliation artifact.

Invalid version, kind, repository cardinality, action type, symbolic ref, OID, or sensitive serialized value fails before Git reference reads or mutation.

Artifact validation failure returns exit code `1`. The artifact must not be repaired or partially interpreted by the executor.

## Stale-plan failures

Immediately before mutation, the executor re-resolves every planned source and target ref.

If any current OID differs from its planned value, or any required ref cannot be resolved:

- no action in the execution is attempted;
- the offending action is marked failed and stale internally;
- all other actions remain skipped;
- apply returns exit code `1`;
- recovery requires a new status/plan cycle.

A force-with-lease push is an additional remote-side target guard. It does not authorize replay of a stale plan.

## Mutation failures

Mutation executes in deterministic plan order and stops after the first failed Git operation.

The executor preserves:

- successful earlier actions as `APPLIED`;
- the failing action as `FAILED`;
- later unattempted actions as `SKIPPED`;
- resulting OIDs for successful actions where available;
- a sanitized diagnostic for the failed action.

There is no rollback or cross-action transaction. Recovery starts by re-observing current state and producing a new plan.

## Multi-repository aggregation

Configured repositories execute independently and may complete out of order. Aggregate output is restored to configuration order.

When one repository execution fails:

- successful repository results remain in output;
- the failed repository result includes its error;
- JSON output remains a complete valid document;
- human output shows the repository error even when no action was constructed;
- the aggregate diagnostic selects the first failed repository in configuration order;
- the process returns exit code `1`.

This preserves diagnostic evidence without allowing automation to interpret partial failure as success.

## Dry-run

Dry-run follows the same observation and planning path as real apply but performs no executor mutation.

Dry-run may still fail because configuration, observation, branch/OID resolution, topology, or planning is invalid.

Current behavior has one inconsistency: non-dry apply rejects ahead or diverged state before planning and returns `2`, while dry-run reaches the planner and currently returns `1` when force authorization is absent. The CLI error taxonomy should normalize this without weakening the planner's fail-closed behavior.

Dry-run output is preview evidence, not proof that later apply remains safe. Real apply repeats stale-ref validation.

## Force behavior

`--force` currently authorizes overwrite planning for ahead or diverged default-branch mirrors. The actual mutation uses force-with-lease.

Force does not bypass:

- configuration validation;
- artifact validation;
- source or target stale checks;
- lease validation;
- Git execution failures;
- process failure status.

The force path is transitional until explicit branch/ref policy exists.

## Journal failures

Merged `main` can construct journal records in memory but does not yet require or persist them during apply.

When persistence becomes part of apply, the required behavior is:

1. persist pre-mutation intent before the first mutation;
2. fail closed if required intent persistence fails;
3. attempt to persist final or failure outcomes after execution;
4. surface safe relative journal references;
5. return non-zero when required evidence cannot be written.

A journal failure must never be silently ignored for a mutation path that claims audited execution.

## Output failures

Failure to serialize or write requested JSON output returns `1`, even if repository work completed. The CLI must not emit a second partial JSON document.

Human diagnostics belong on stderr. Structured command results belong on stdout.

## Retry rules

Safe retry follows this sequence:

1. inspect the returned repository and action results;
2. rerun status against current remotes;
3. produce a new plan from current observations;
4. review destructive or policy-relevant changes again;
5. apply the new plan.

Do not retry by replaying an old in-memory plan, editing expected OIDs, or weakening lease/stale checks.

## Test obligations

Changes to reconciliation must test the relevant failure boundary:

- invalid input causes zero Git operations;
- stale later actions prevent action zero from mutating;
- partial runtime failure preserves applied/failed/skipped order;
- failed repository execution produces non-zero process status;
- mixed repository results remain ordered and machine-readable;
- errors do not expose credentials, tokens, or unnecessary absolute paths;
- retries re-plan from current state.
