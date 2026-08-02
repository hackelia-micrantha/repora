# Current implementation plan

Status: Active

## Release objective

The next meaningful Repora release is a coherent, local-first Git mirror controller with one trustworthy observation, planning, execution, and evidence path.

The release is intentionally narrower than the broader repository-control-plane vision.

## Required v0.1 capability

A v0.1-quality controller must:

- load durable repository topology from strict configuration;
- resolve runtime transport without persisting credentials or derived URLs as identity;
- observe one canonical and one mirror default branch;
- classify equal, behind, ahead, and diverged state;
- produce one deterministic executable plan artifact;
- render the exact executable plan for review;
- reject invalid or stale plans before mutation;
- enforce explicit destructive-change policy;
- execute with lease protection;
- return detailed partial results and correct process status;
- persist durable pre/post execution evidence;
- ship through a documented, repeatable binary release flow.

## Current state

| Area | State | Notes |
| --- | --- | --- |
| Durable `uid` identity | Complete | Cache and durable artifacts use logical identity rather than location. |
| Provider/path topology | Complete for built-in GitHub/GitLab HTTPS | Configurable bases and transport selection remain outside the current slice. |
| Status classification | Complete for one mirror/default branch | Short commit evidence failures are not yet surfaced consistently. |
| Planner/executor separation | Complete | Dry-run and apply share the internal observation-to-plan path. |
| Stale-plan validation | Complete | Complete action preflight occurs before action zero. |
| Public CLI JSON envelopes | Complete for status, plan, and apply | Versioned schemas and representative golden contracts exist. |
| Executable plan artifact | Partially complete | Schema, validation, conversion, and executor consumption exist; the public `plan` command does not yet emit the exact artifact. |
| Execution results | Partially complete | Executor preserves detailed outcomes; public apply results still collapse them. |
| Apply exit semantics | Complete | Repository execution failures preserve output and return non-zero. |
| Journal record model | Partially complete | Versioned records and executor projection exist; apply integration and required persistence do not. |
| Branch/ref policy | Not started | Current force flag remains transitional. |
| Multi-mirror runtime | Deferred until policy/evidence gates | Read-side status must precede apply expansion. |
| Release packaging | Not started | CI builds verification binaries but does not publish supported releases. |

## Immediate sequence

### 1. Establish documentation authority

Exit condition:

- one index identifies current, proposed, and historical sources;
- one current-system document matches merged source;
- failure and recovery semantics are explicit;
- one active plan replaces duplicate roadmap/checklist interpretation;
- ADR lifecycle and supersession rules are documented.

### 2. Make review and execution share one artifact

Exit condition:

- `repoctl plan` renders or exports the exact validated reconciliation artifact;
- apply accepts that artifact boundary without rebuilding intent;
- public preview and executable action types no longer represent independent decisions;
- forced actions and OID preconditions are reviewable without exposing secrets.

### 3. Integrate execution evidence vertically

Exit condition:

- pre-mutation intent is persisted before action zero;
- final applied, failed, stale, and skipped outcomes are persisted;
- required journal write failure is fail-closed;
- human and JSON output expose a safe relative journal reference;
- retention and filesystem ownership are documented.

### 4. Define explicit ref policy

Exit condition:

- default behavior remains default-branch-only and deny-by-default;
- tags and non-default refs are explicitly governed;
- protected refs and force behavior are policy inputs;
- planner explains allowed, skipped, and rejected refs;
- executor rejects policy-invalid artifacts defensively.

### 5. Expose detailed execution outcomes

Exit condition:

- public apply results preserve action-level applied/failed/skipped/stale state;
- before, desired, and resulting OIDs are available where safe;
- partial success and retry guidance are machine-readable;
- compatibility changes use an explicit JSON contract version.

### 6. Expand read-side topology

Exit condition:

- status evaluates every configured mirror independently;
- one mirror failure does not hide other results;
- aggregate status semantics are deterministic;
- stable mirror targeting identity is defined before mutation expansion.

### 7. Add multi-mirror apply

Exit condition:

- actions and results are per mirror;
- explicit targeting is unambiguous;
- no cross-remote atomicity is implied;
- partial failure is journaled and retry-safe through re-planning.

### 8. Package v0.1

Exit condition:

- supported platforms are explicit;
- tagged binaries and checksums are published;
- packaged binaries receive smoke coverage;
- release permissions remain least privilege;
- installation, verification, compatibility, and release checklist are documented.

## Explicit deferrals

The following tracks are valid but are not on the current release critical path:

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
- Add generic abstractions only after at least two implemented consumers demonstrate the shared behavior.
- Keep compatibility serializers as views, never decision authorities.
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
