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
- produce one deterministic exact executable artifact across required mirrors;
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
| Exact multi-mirror planning | Complete | Artifact v2 actions are identity-matched and deterministic. |
| Runtime target rebinding | Complete | Imported targets bind by provider/path to current local aliases; serialized aliases and positions are not authority. |
| All-target preflight | Complete | Topology, policy, branches, and every expected OID are validated before action zero. |
| Audited multi-mirror dry-run | Complete | One repository-level intent/result pair preserves every validated, stale, failed, or skipped action. |
| Execution record v3 | Complete | Path-bound source/target evidence; v1/v2 remain parseable. |
| Real multi-mirror mutation | Active next gate | Independent continuation and versioned per-target apply results remain. |
| Release packaging | Not started | CI builds verification binaries but does not publish supported releases. |

## Immediate sequence

### 1. Add independent ordered multi-mirror mutation

Completed foundation:

- exact artifact v2 binds every reviewed target to provider/path;
- status and planning match targets by identity rather than order;
- imported artifacts bind targets to current runtime aliases without rewriting intent;
- configuration, status, policy, force intent, and default branches are validated before intent;
- executor preflight validates every expected source and target OID before action zero;
- audited dry-run writes execution-record v3 intent/result evidence;
- real multi-mirror mutation remains explicitly gated.

Remaining exit condition:

- publish a versioned per-target apply output contract;
- make convenience apply and `--plan-file` use one multi-target execution path;
- execute actions sequentially in deterministic artifact order;
- continue later independent mirrors after one runtime push fails;
- preserve `APPLIED`, `FAILED`, `SKIPPED`, and `STALE` per target;
- persist path-bound before/desired/after/outcome details in one repository-level result record;
- return nonzero when any mirror fails without hiding successful outcomes;
- use force-with-lease for each action already marked forced;
- imply no rollback or cross-remote atomicity;
- require fresh status and re-planning for retry.

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
- Reuse the audited dry-run preflight unchanged for real execution.
- Do not imply cross-remote atomicity.

## Definition of done

A slice is complete only when observable behavior, failure/recovery tests, versioned contracts, architecture updates, issue acceptance criteria, and CI/security validation are complete.

## Maintenance

Update this plan when a release gate completes, changes order, or is explicitly deferred.
