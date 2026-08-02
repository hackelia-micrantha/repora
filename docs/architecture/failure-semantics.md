# Failure and recovery semantics

Status: Current

Repora controls durable Git state. Failures must therefore be explicit in human output, machine-readable output, and process exit status.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | The requested operation completed successfully. For dry-run, validation and stale-ref preflight completed without mutation. |
| `1` | Configuration, observation, planning, artifact validation, stale input, execution, serialization, or other operational failure. |
| `2` | A real mutation requires explicit destructive authorization, such as an ahead or diverged mirror without `--force`. |

A command must not return `0` merely because it successfully serialized or printed a failed result.

## Processing stages

```text
load configuration
  -> observe repository state
  -> classify relationship
  -> build exact plan artifact
  -> validate configuration, state, scope, and authorization
  -> stale-ref preflight
  -> mutate when not dry-run
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

Observation includes cache preparation, remote configuration, fetch, remote HEAD setup, divergence classification, and commit evidence resolution.

Without `--continue-on-error`, observation still completes through the bounded worker set, but normal command output is suppressed after any repository observation fails and the command returns `1`.

With `--continue-on-error`:

- independently observed repositories remain available for status or compatibility plan output;
- failed repository observations are omitted from normal result collections;
- status and compatibility plan return non-zero;
- an exact executable artifact is not emitted when selected planning is incomplete;
- exact artifact apply refuses partial observation.

Observation failure and destructive-state refusal are different conditions. Current status/plan aggregation may return `2` when both occur because the unsafe-state code has precedence. This can be revisited when the broader CLI error taxonomy is stabilized.

## Planning failures

Planning must be pure with respect to remotes and must describe destructive intent independently of mutation authorization.

A planning failure:

- performs no mutation;
- returns the repository identity and available compatibility result context where possible;
- suppresses exact artifact export when any selected repository plan is incomplete;
- returns exit code `1`;
- requires correction or a fresh observation before retry.

Planning fails closed for unsupported state, ambiguous topology, missing default branches, missing OIDs, or invalid artifact construction. Ahead and diverged state produce an explicit forced action; `--force` is checked only when real mutation is requested.

## Artifact validation failures

The executor accepts only a validated versioned reconciliation artifact.

Invalid version, kind, repository identity, action type, symbolic ref, OID, sensitive serialized value, or repository cardinality fails before mutation.

Imported v1 artifacts are additionally checked against current configuration and observation:

- durable `uid` must identify a configured repository;
- canonical and mirror provider/remote ownership must match configuration;
- each repository may contain at most one action;
- action branches must match current canonical and mirror default branches;
- current `EQUAL` state requires no action;
- current `BEHIND` state requires one non-forced action;
- current `AHEAD` or `DIVERGED` state requires one forced action.

A mismatch is treated as stale or policy-invalid input and returns exit code `1`. The artifact must not be repaired or partially interpreted by the executor.

## Stale-plan failures

Dry-run and real apply re-resolve every planned source and target ref after current repository observation.

If any current OID differs from its planned value, or any required ref cannot be resolved:

- no action in the repository execution is attempted;
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

Configured repositories execute independently and may complete out of order. Aggregate output is restored to configuration or artifact order.

When one repository execution fails:

- successful repository results remain in output;
- the failed repository result includes its error;
- JSON output remains a complete valid document;
- human output shows the repository error even when no action was constructed;
- the aggregate diagnostic selects the first failed repository in deterministic order;
- the process returns exit code `1`.

This preserves diagnostic evidence without allowing automation to interpret partial failure as success.

## Dry-run

Dry-run uses the same exact artifact boundary as real apply and performs no mutation.

Convenience dry-run observes current state, builds the artifact, validates configuration and current-state intent, validates current default-branch scope, and performs source/target stale-ref preflight.

Imported-artifact dry-run performs the same checks without rebuilding intent. A forced action is visible and preflighted without requiring `--force`, because no mutation is authorized or attempted.

Dry-run may still fail because configuration, observation, artifact structure, current state, default-branch scope, or OID preconditions are invalid. A successful dry-run is current validation evidence, not a durable guarantee; real apply repeats current observation and stale-ref preflight.

## Force behavior

`--force` authorizes real mutation for an exact action already marked `force: true`. It does not cause the planner to invent or hide destructive intent.

Actual destructive mutation uses force-with-lease against the artifact's observed target OID.

Force does not bypass:

- configuration validation;
- artifact validation;
- current-state/action consistency;
- default-branch scope validation;
- source or target stale checks;
- lease validation;
- Git execution failures;
- process failure status.

The force flag remains a transitional authorization mechanism until explicit branch/ref policy exists.

## Journal failures

The journal package can construct and persist validated records, but apply does not yet require or expose pre/post execution records.

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
3. produce a new exact artifact from current observations;
4. review destructive or policy-relevant changes again;
5. apply the new artifact.

Do not retry by editing expected OIDs, weakening force/lease/stale checks, or replaying an artifact after validation reports drift.

## Test obligations

Changes to reconciliation must test the relevant failure boundary:

- invalid input causes zero Git operations;
- current state and artifact intent must agree;
- non-default or excess actions fail before stale-ref reads;
- dry-run performs stale-ref preflight without mutation;
- stale later actions prevent action zero from mutating;
- partial runtime failure preserves applied/failed/skipped order;
- failed repository execution produces non-zero process status;
- mixed repository results remain ordered and machine-readable;
- errors do not expose credentials, tokens, or unnecessary absolute paths;
- retries re-plan from current state.
