# Failure and recovery semantics

Status: Current

Repora controls durable Git state. Failures must be explicit in human output, machine-readable output, journal evidence where applicable, and process exit status.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Requested operation completed successfully. |
| `1` | Configuration, observation, planning, artifact, stale input, journal, execution, serialization, or other operational failure. |
| `2` | Observation/planning is complete but unsafe state or a real destructive mutation requires explicit authorization. |

Operational failure takes precedence over unsafe-state reporting because incomplete observation cannot prove the rest of the topology safe.

## Configuration failures

Strict configuration validation occurs before Git work.

Failures include unsupported providers/modes/policies, invalid paths, duplicate mirror targets, no mirrors, credential-bearing URLs, and legacy URLs in a multi-mirror entry.

A configuration failure emits no normal result, performs no Git operation, and returns `1`.

## Status observation

### Canonical failure

Cache preparation, canonical configuration/fetch/HEAD, and canonical commit evidence are shared per repository. Failure at this boundary makes the repository result incomplete and returns `1`.

Without `--continue-on-error`, a repository-level failure suppresses normal status output as in the historical single-mirror behavior. With continuation, other repositories remain available.

### Mirror failure

Multi-mirror status observes each mirror independently after canonical succeeds.

A mirror-specific resolution, configuration, fetch, HEAD, comparison, or commit-evidence failure:

- retains the stable `provider:path` target;
- emits state `ERROR` and a target-local diagnostic;
- does not hide or prevent observation of later mirrors;
- makes the command return `1`.

If every mirror observation is complete, any `AHEAD` or `DIVERGED` result returns `2`; otherwise status returns `0`.

## Multi-mirror planning

`repoctl plan` supports one or more mirrors and matches status results to configuration by stable target identity rather than array position.

Planning fails with exit `1` when:

- canonical or any mirror observation is incomplete;
- status repository identity or mirror cardinality is inconsistent;
- a configured target is missing or duplicated in the status result;
- provider/path identity cannot be derived safely;
- policy, default branch, or OID evidence cannot be resolved;
- the generated artifact fails validation.

`plan --artifact` emits no partial artifact when any selected repository is incomplete, including with `--continue-on-error`. An exact artifact must represent the complete selected intent.

When planning succeeds and the artifact contains ahead/diverged forced intent, the artifact is still emitted for review and the command returns `2` unless `--force` is supplied.

Human multi-mirror plan output uses stable `provider:path` targets. The historical `plan --json` v1 compatibility view is rejected for multi-mirror topology with exit `1`; operators must use `--artifact` for the exact machine-readable plan.

## Mutation topology gate

Apply and sync currently require exactly one configured mirror. A multi-mirror repository is rejected before reconciliation observation with exit `1`. The CLI never chooses the first mirror implicitly.

## Artifact failures

Invalid kind/version/UID/topology/path/action/ref/OID values, policy mismatch, or state/action/force mismatch fail before mutation with exit `1`.

Version-2 path mismatch is rejected before repository Git reads. Version-1 imports remain limited to their historical single-mirror provider/alias contract.

Imported artifacts are not repaired, reordered, retargeted, or partially reinterpreted.

## Stale preflight

Current single-mirror dry-run and real apply resolve every planned source and target OID after observation. If any ref is missing or differs:

- no action is attempted;
- the offending action is `STALE`/failed internally;
- other actions remain skipped;
- result evidence is attempted;
- the command returns `1`;
- retry requires fresh status and planning.

Force-with-lease is additional defense and does not authorize stale replay.

The next multi-mirror execution slice must complete this validation for every target before action zero.

## Force behavior

The closed ref policy records ahead/diverged reconciliation as forced intent. `--force` authorizes only a real action already marked forced; it does not alter the plan.

For planning, `--force` acknowledges destructive intent for process-status purposes only. It does not mutate or weaken the artifact.

Dry-run may review and stale-check forced intent without authorizing mutation.

Force never bypasses configuration, policy, artifact, state, default-branch, OID, lease, journal, or Git failures.

## Journal failures

Apply and dry-run require one immutable intent/result pair per repository execution.

- intent persistence occurs before executor preflight can reach mutation;
- intent-write failure performs zero mutation and returns `1`;
- stale or runtime failure still attempts result persistence;
- result-write failure returns `1` even if mutation completed;
- available safe execution ID and references remain in output;
- a present intent without a result requires reconciliation against current Git state before retry.

Multi-mirror planning does not create journal evidence because it does not enter the execution boundary.

Journals are evidence, never replay authority.

## Runtime mutation failure

The current single-mirror executor preserves action outcomes and returns nonzero. There is no rollback.

Future multi-mirror execution must define continuation after one remote fails, preserve each target outcome, and avoid atomicity claims.

## Multi-repository aggregation

Repository tasks may complete out of order; output is restored to configuration or artifact order.

Successful status results remain visible when another repository fails. Exact plan artifacts are stricter: incomplete selected planning suppresses the artifact rather than emitting a partial executable document.

## Output failure

Failure to serialize or write requested JSON returns `1` even if earlier Git reads completed. The CLI must not emit a second partial JSON document.

Human diagnostics belong on stderr; structured results belong on stdout.

## Retry rules

1. inspect status/apply/journal evidence;
2. observe all current targets again;
3. build a new exact artifact;
4. review destructive intent again;
5. apply only through the supported execution boundary.

Do not edit expected OIDs, weaken policy/lease checks, infer identity from mirror position, or replay an artifact after drift is reported.

## Test obligations

Changes must test the applicable boundary:

- invalid input performs zero mutation;
- one mirror failure does not hide later status results;
- operational status failure overrides unsafe exit reporting;
- observations are matched by stable target identity rather than order;
- incomplete multi-mirror planning suppresses exact artifact output;
- destructive plan intent remains visible and returns deterministic status;
- multi-mirror apply/sync remain gated before observation;
- state, policy, artifact intent, default branches, and OIDs agree before action zero;
- dry-run performs complete stale preflight without mutation;
- intent-write failure prevents mutation;
- result-write failure remains nonzero;
- partial outcomes remain ordered and machine-readable;
- diagnostics exclude secrets and unnecessary absolute paths;
- retry re-plans from current state.
