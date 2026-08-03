# Current implementation plan

Status: Active

## Release objective

The next meaningful Repora release is a coherent, local-first Git mirror controller with one trustworthy observation, planning, execution, policy, and evidence path.

## Required v0.1 capability

A v0.1-quality controller must:

- load durable repository topology from strict configuration;
- resolve runtime transport without persisting credentials or URLs as identity;
- observe one canonical and one or more mirror default branches;
- classify state and failures independently per mirror;
- produce one deterministic exact executable artifact;
- reject invalid or stale plans before mutation;
- enforce closed reference and destructive-change policy;
- execute with lease protection;
- preserve per-mirror partial outcomes and correct process status;
- persist durable intent/result evidence;
- ship through a documented repeatable binary release flow.

## Current state

| Area | State | Notes |
| --- | --- | --- |
| Durable `uid` identity | Complete | Cache and durable artifacts use logical identity. |
| Provider/path topology | Complete for current providers | Multiple mirrors require unambiguous provider/path targets. |
| Ref policy v1 | Complete | Default-branch-only and require-force; unsupported expansion is rejected. |
| Multi-mirror status | Complete | Ordered observations, stable `provider:path` identity, mirror-local failures, and status v2. |
| Planner/executor separation | Complete | The executor does not recompute reconciliation policy. |
| Exact executable plan artifact | Complete for path-bound single-mirror execution | New plans emit v2 with canonical/mirror provider paths; v1 remains historical single-mirror compatibility. |
| Execution journals | Complete for one-mirror apply/dry-run | Intent failure is fail-closed; result-write failure is non-zero; records may reference plan v1 or v2. |
| Multi-mirror apply | Active next gate | Target identity foundation is complete; multi-action planning, preflight, outcomes, and evidence remain. |
| Release packaging | Not started | CI builds verification binaries but does not publish supported releases. |

## Immediate sequence

### 1. Complete exact multi-mirror apply

Completed foundation:

- reconciliation artifact v2 binds source and target refs to provider-relative paths;
- version-2 topology mismatch fails before Git reads;
- configuration order and runtime aliases are not durable target identity;
- version-1 plans remain parseable only for historical single-mirror execution.

Remaining exit condition:

- build one exact action set across all non-equal mirrors;
- map each reviewed provider/path target to its current runtime alias after configuration reordering;
- complete all topology, policy, default-branch, and OID checks before action zero;
- make convenience apply and `--plan-file` use the same multi-target path;
- continue later independent mirror attempts after a runtime failure;
- expose per-mirror outcomes in versioned apply output;
- persist per-mirror path and outcome evidence in one repository-level journal intent/result pair;
- make retry require fresh status and re-planning;
- make no cross-remote atomicity or rollback claim.

### 2. Package v0.1

Exit condition:

- supported platforms are explicit;
- tagged binaries and checksums are published;
- packaged binaries receive smoke coverage;
- release permissions remain least privilege;
- installation, verification, compatibility, and release checklist are documented.

## Explicit deferrals

- tags, non-default branches, wildcard refspecs, and deleted-ref reconciliation;
- concurrent mirror mutation in the first multi-mirror implementation;
- managed repository artifacts and README templating;
- advanced document routing and repository assessments;
- optional Anthesis policy integration;
- provider provisioning or hosted control-plane behavior;
- generalized cross-domain diff engines;
- automated rollback or cross-repository/cross-remote transactions.

Deferred tracks must reuse the core plan, policy, execution, and evidence substrate rather than introduce parallel mutation paths.

## Simplicity constraints

- Prefer vertical capabilities over disconnected internal models.
- Do not add a package solely to match an architecture diagram.
- Add generic abstractions only after multiple implemented consumers prove shared behavior.
- Keep compatibility serializers as views, never decision authorities.
- Treat configuration order and runtime aliases as execution details, never durable mirror identity.
- Keep ref-policy v1 closed.
- Do not imply cross-remote atomicity.

## Definition of done

A slice is complete only when observable behavior, failure/recovery tests, versioned contracts, architecture updates, issue acceptance criteria, and CI/security validation are complete.

## Maintenance

Update this plan when a release gate completes, changes order, or is explicitly deferred.
