# Current implementation plan

Status: Active

## Release objective

The next meaningful Repora release is a coherent, local-first Git mirror controller with one trustworthy observation, planning, execution, policy, and evidence path.

The release is intentionally narrower than the broader repository-control-plane vision.

## Required v0.1 capability

A v0.1-quality controller must:

- load durable repository topology from strict configuration;
- resolve runtime transport without persisting credentials or derived URLs as identity;
- observe one canonical and one or more mirror default branches;
- classify equal, behind, ahead, and diverged state per mirror;
- produce one deterministic executable plan artifact;
- export and execute that exact artifact;
- reject invalid or stale plans before mutation;
- enforce explicit closed reference and destructive-change policy;
- execute with lease protection;
- return detailed partial results and correct process status;
- persist durable pre/post execution evidence;
- ship through a documented, repeatable binary release flow.

## Current state

| Area | State | Notes |
| --- | --- | --- |
| Durable `uid` identity | Complete | Cache and durable artifacts use logical identity rather than location. |
| Provider/path topology | Complete for built-in GitHub/GitLab HTTPS | Configurable bases and transport selection remain outside the current slice. |
| Status classification | Complete for one mirror/default branch | Classification and required commit evidence fail together. |
| Planner/executor separation | Complete | Planning is deterministic and the executor does not recompute policy. |
| Exact executable plan artifact | Complete for one mirror/default branch | `plan --artifact` exports the exact artifact; `apply --plan-file` consumes it without rebuilding intent. |
| Stale-plan validation | Complete | Complete action preflight occurs before action zero. |
| Public CLI JSON envelopes | Complete for current status, compatibility plan, apply, and execution records | Contract changes require explicit versions and retained historical schemas. |
| Apply exit semantics | Complete | Repository execution failures preserve output and return non-zero. |
| Execution journals | Complete for current apply/dry-run | Immutable intent/result entries are required; intent failure is fail-closed and result-write failure is non-zero. |
| Branch/ref policy | Complete for version 1 | Omitted policy normalizes to default-branch-only and require-force; unsupported expansion is rejected. |
| Multi-mirror status | Not started | Read-side observation and stable target identity are the next gate. |
| Multi-mirror apply | Blocked | Must follow the read-side model and preserve per-mirror evidence. |
| Release packaging | Not started | CI builds verification binaries but does not publish supported releases. |

## Immediate sequence

### 1. Expand read-side topology

Exit condition:

- configuration accepts unambiguous multiple mirror targets;
- status evaluates every configured mirror independently;
- one mirror failure does not hide other results;
- aggregate status semantics are deterministic;
- stable provider/path mirror identity is defined;
- JSON contract changes use an explicit version;
- plan/apply remain explicitly gated until mutation support exists.

### 2. Add multi-mirror apply

Exit condition:

- exact artifacts identify every target unambiguously;
- complete policy and stale-ref preflight occurs before action zero;
- actions, results, and journal evidence are per mirror;
- later independent mirrors may still be attempted after a runtime failure;
- no cross-remote atomicity or rollback is implied;
- retry requires fresh observation and re-planning.

### 3. Package v0.1

Exit condition:

- supported platforms are explicit;
- tagged binaries and checksums are published;
- packaged binaries receive smoke coverage;
- release permissions remain least privilege;
- installation, verification, compatibility, and release checklist are documented.

## Explicit deferrals

The following tracks are valid but are not on the current release critical path:

- tags, non-default branches, wildcard refspecs, and deleted-ref reconciliation;
- managed repository artifacts and README templating;
- advanced document routing, trust tiers, receipts, summaries, and AST selectors;
- repository assessment and evidence reports;
- optional Anthesis policy integration;
- provider provisioning or hosted control-plane behavior;
- generalized file/workflow/artifact diff engines;
- automated rollback or cross-repository transactions.

These tracks must reuse the core plan, policy, execution, and evidence substrate. They must not create parallel mutation paths.

## Simplicity constraints

- Prefer vertical capabilities over long sequences of disconnected internal models.
- Do not add a package solely to match an architecture diagram.
- Unify versioned envelopes and safety conventions, not unrelated domain semantics.
- Add generic abstractions only after at least two implemented consumers demonstrate shared behavior.
- Keep compatibility serializers as views, never decision authorities.
- Keep ref-policy v1 closed until runtime and artifact contracts intentionally version broader scope.
- Use GitHub issues for task state; do not duplicate every checkbox here.

## Definition of done

A slice is complete only when:

- observable behavior is implemented;
- failure and recovery behavior are tested;
- public contracts are versioned when changed;
- architecture and current limitations are updated;
- the owning issue acceptance criteria are satisfied or explicitly revised;
- CI passes without weakening safety assertions.

## Maintenance

Update this plan when a release gate completes, changes order, or is explicitly deferred. Historical plans should remain available for context but must not be presented as current implementation state.
