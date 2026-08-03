# Failure and recovery semantics

Status: Current

Repora controls durable Git state. Failures must be explicit in human output, versioned machine-readable output, journal evidence, and process exit status.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Requested operation completed successfully. |
| `1` | Configuration, observation, planning, artifact, stale input, journal, execution, serialization, or other operational failure, including partial success. |
| `2` | Complete destructive intent requires explicit command authorization. |

Operational failure takes precedence over unsafe-state reporting.

## Preparation failures

Before any selected repository mutates, Repora completes observation and exact-artifact preparation for every selected repository.

Configuration, observation, artifact version/cardinality, planning, topology, or serialization failure:

- performs no selected mutation;
- writes no intent;
- returns `1`;
- emits valid apply v3 `ERROR` results for affected repositories when structured output was requested.

Exact artifact export is similarly suppressed when selected planning is incomplete.

## Force authorization

Ahead or diverged state creates forced intent in the artifact. A real selected execution containing any forced action requires `--force` before audit initialization or mutation.

Missing authorization:

- performs no mutation;
- writes no intent;
- returns `2`.

The flag authorizes only actions already marked forced and never changes the plan.

## Repository preflight

Before intent persistence, Repora validates durable identity, canonical and mirror provider/path topology, current status, state/action/force agreement, and current default branches.

Failure here writes no intent, performs no mutation for that repository, and returns `1`.

After intent persistence, every expected source and target OID is checked before action zero. If any action is stale:

- no action in that repository is attempted;
- earlier and later unattempted actions remain `SKIPPED`;
- the offending action is `STALE`;
- result persistence is attempted;
- the command returns `1`.

Runtime aliases are derived from current provider/path configuration. Serialized aliases and positions cannot retarget imported intent.

## Independent runtime mutation

After complete repository preflight, mirror actions execute sequentially in exact artifact order.

If one runtime push fails:

- that action becomes `FAILED`;
- later independent actions are still attempted;
- successful actions become `APPLIED` with their resulting OID;
- earlier success is not rolled back;
- all available outcomes are preserved in apply v3 and execution-record v3;
- the command returns `1`.

A result such as `APPLIED, FAILED, APPLIED` is valid. It is partial success, not a transaction.

## Journal failures

Each repository execution requires one immutable intent/result pair.

- intent-write failure prevents every push in that repository and returns `1`;
- stale or runtime failure still attempts result persistence;
- result-write failure returns `1` even if one or more pushes completed;
- safe execution ID and available references remain in output;
- an intent without a result requires reconciliation against current Git state before retry.

Path-bound artifact v2 uses execution-record v3. Historical records remain parseable. Journals are evidence, never replay authority.

## Multi-repository aggregation

All selected repositories prepare before mutation. After preparation and force authorization, repository executions may run concurrently and are independent.

One repository failure does not erase outcomes from other repositories. Output is restored to configuration or artifact order. There is no cross-repository rollback or transaction.

## Output contracts

- single-mirror-only selections retain apply v2;
- mixed or multi-mirror selections use apply v3;
- apply v3 exposes per-target before, desired, after, outcome, and sanitized error;
- structured serialization failure returns `1` even if Git work completed;
- human diagnostics belong on stderr and requested JSON belongs on stdout.

## Retry rules

After stale or partial failure:

1. inspect apply and execution-record evidence;
2. observe every current target again;
3. build a new exact artifact;
4. review destructive intent again;
5. execute the new artifact.

Do not edit expected OIDs, infer identity from position, replay journal evidence, or attempt an automatic rollback.

## Test obligations

Changes must test:

- preparation or authorization failure performs zero mutation and writes no intent;
- all expected refs validate before action zero;
- alias reordering cannot retarget an action;
- a stale later action yields skipped/stale evidence and zero pushes;
- runtime failure of a middle mirror does not prevent a later mirror push;
- force-with-lease uses the reviewed target OID;
- apply v3 and execution-record v3 preserve ordered partial outcomes;
- intent and result persistence failures remain nonzero;
- real local bare mirrors reproduce partial success and recovery evidence;
- diagnostics exclude secrets and absolute paths;
- retry requires fresh planning.
