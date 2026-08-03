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

## Configuration and observation failures

Strict configuration validation occurs before Git work. Unsupported providers, modes, policies, invalid paths, duplicate targets, no mirrors, credential-bearing URLs, and ambiguous multi-mirror legacy URLs return `1`.

Canonical failure makes the whole repository observation incomplete. Mirror-local failure retains the stable `provider:path` result as `ERROR`, does not hide later mirrors, and returns `1`.

## Multi-mirror planning

`repoctl plan` supports one or more mirrors and matches status results to configuration by stable target identity rather than array position.

Planning returns `1` when observation, identity, policy, default-branch, OID, or generated-artifact evidence is incomplete. `plan --artifact` never emits a partial executable artifact. Complete forced intent remains visible and returns `2` unless acknowledged with `--force`.

The historical `plan --json` v1 compatibility view is rejected for multi-mirror topology; use `--artifact`.

## Audited multi-mirror dry-run

`apply|sync --dry-run` supports one or more mirrors. It accepts a freshly built exact artifact or `--plan-file` artifact v2.

Before journal intent, Repora validates:

- durable repository identity;
- configured canonical provider/path;
- every configured mirror target exactly once;
- complete current status evidence;
- state/action/force agreement under ref-policy v1;
- current canonical and mirror default branches through runtime-bound aliases.

Failure at this boundary performs no mutation, writes no journal intent, and returns `1`.

After intent persistence, executor preflight checks every expected source and target OID before action zero. If any target is missing or changed:

- no action is attempted;
- earlier actions remain `SKIPPED`;
- the offending action is `STALE`;
- later actions remain `SKIPPED`;
- execution-record v3 result persistence is attempted;
- the command returns `1`;
- retry requires complete re-observation and a new exact artifact.

Runtime aliases are resolved from current `provider:path` configuration. Serialized aliases and array positions cannot retarget an imported artifact.

Multi-mirror dry-run human output is supported. Multi-mirror `--json` returns `1` until a versioned per-target apply result contract is published.

## Real mutation topology gate

Real apply and sync currently require exactly one configured mirror. A multi-mirror repository without `--dry-run` is rejected before reconciliation observation with exit `1`. The CLI never chooses the first mirror implicitly.

## Artifact failures

Invalid kind/version/UID/topology/path/action/ref/OID values, unknown targets, duplicate actions, policy mismatch, or state/action/force mismatch fail closed with exit `1`.

Artifact v2 path mismatch is rejected before repository ref reads. Artifact v1 remains limited to historical single-mirror provider/alias execution. Imported artifacts are not repaired, reordered, retargeted, or partially reinterpreted.

## Force behavior

The closed ref policy records ahead/diverged reconciliation as forced intent. `--force` authorizes only a real action already marked forced; it does not alter the plan.

Dry-run may review and stale-check forced intent without authorizing mutation. Force never bypasses topology, policy, branch, OID, lease, journal, or Git failures.

## Journal failures

Apply and dry-run require one immutable intent/result pair per repository execution.

- intent-write failure performs zero mutation and returns `1`;
- stale or runtime failure still attempts result persistence;
- result-write failure returns `1` even if mutation completed;
- available safe execution ID and references remain in output;
- a present intent without a result requires reconciliation against current Git state before retry.

Path-bound plan v2 uses execution-record v3 with provider paths. Historical v1/v2 records remain parseable. Journals are evidence, never replay authority.

## Runtime mutation failure

The current real mutation path remains single-mirror and returns nonzero without rollback.

The next multi-mirror mutation slice must continue independent later actions after one remote fails, preserve every target outcome, and avoid atomicity claims.

## Multi-repository aggregation

Repository tasks may complete out of order; output is restored to configuration or artifact order. One repository failure does not erase available results from other repositories.

Exact plan artifacts are stricter: incomplete selected planning suppresses the complete artifact rather than emitting a partial executable document.

## Output failure

Failure to serialize or write requested structured output returns `1`. Human diagnostics belong on stderr; structured results belong on stdout.

## Retry rules

1. inspect status, dry-run/apply, and journal evidence;
2. observe every current target again;
3. build a new exact artifact;
4. review destructive intent again;
5. execute only through the supported boundary.

Do not edit expected OIDs, weaken policy or lease checks, infer identity from mirror position, or replay an artifact after drift is reported.

## Test obligations

Changes must test the applicable boundary:

- invalid input performs zero mutation;
- one mirror failure does not hide later status results;
- observations and imported actions are matched by stable target identity;
- incomplete planning suppresses exact artifact output;
- real multi-mirror mutation remains gated before observation;
- multi-mirror dry-run validates every target before action zero;
- alias reordering cannot retarget an artifact;
- a stale later target produces ordered skipped/stale evidence and zero mutation;
- intent-write failure prevents mutation;
- result-write failure remains nonzero;
- diagnostics exclude secrets and unnecessary absolute paths;
- retry re-plans from current state.
