# Current implementation plan

Status: Active

## Release objective

The next meaningful Repora release is a supported v0.1 binary distribution of the completed local-first multi-mirror controller.

## Required v0.1 capability

A v0.1-quality controller must:

- load durable repository topology from strict configuration;
- resolve runtime transport without persisting credentials or URLs as identity;
- observe one canonical and one or more mirror default branches;
- classify state and failures independently per mirror;
- produce one deterministic exact executable artifact;
- reject invalid or stale plans before mutation;
- enforce closed reference and destructive-change policy;
- execute independent mirrors with lease protection and honest partial outcomes;
- persist durable path-bound intent/result evidence;
- ship through a documented repeatable binary release flow.

## Current state

| Area | State | Notes |
| --- | --- | --- |
| Durable `uid` identity | Complete | Cache and durable artifacts use logical identity. |
| Provider/path topology | Complete | Multiple mirrors use unambiguous stable targets. |
| Ref policy v1 | Complete | Default branch only; destructive actions require authorization. |
| Multi-mirror status | Complete | Ordered observations and mirror-local failures in status v2. |
| Exact multi-mirror planning | Complete | Artifact v2 actions are identity-matched and deterministic. |
| Runtime target rebinding | Complete | Provider/path binds to current aliases without rewriting intent. |
| Complete preflight | Complete | Topology, policy, branches, and all expected OIDs validate before action zero. |
| Multi-mirror dry-run | Complete | Path-bound execution-record v3 intent/result evidence. |
| Multi-mirror mutation | Complete | Sequential independent actions continue after runtime failure without rollback. |
| Apply output v3 | Complete | Per-target before/desired/after/outcome evidence and partial-success reporting. |
| Release packaging | Active next gate | CI verifies builds but does not publish supported artifacts. |

## Immediate sequence

### 1. Package v0.1

Exit condition:

- supported operating systems and architectures are explicit;
- tagged binaries are built from a reviewed commit;
- checksums are published beside release assets;
- packaged binaries receive installation and command smoke coverage;
- release workflow permissions remain least privilege;
- artifact naming and version embedding are deterministic;
- installation, checksum verification, compatibility, and rollback guidance are documented;
- a repeatable release checklist identifies required CI/security gates;
- issues #5 and #11 are reconciled or closed through the implementation.

### 2. Post-v0.1 hardening

After packaging, prioritize evidence-backed needs rather than broad abstraction:

- remaining supply-chain CI from #10;
- operator experience found during real multi-mirror use;
- additional ref scope only through a new policy version and ADR;
- provider or transport expansion only with explicit identity/authentication contracts.

## Explicit deferrals

- tags, non-default branches, wildcard refspecs, and deleted-ref reconciliation;
- concurrent mirror mutation;
- automatic rollback or cross-repository/cross-remote transactions;
- managed repository artifacts and README templating;
- advanced document routing and repository assessments;
- optional Anthesis policy integration;
- provider provisioning or hosted control-plane behavior;
- generalized cross-domain diff engines.

Deferred tracks must reuse the current plan, policy, execution, result, and evidence substrate rather than create parallel mutation paths.

## Simplicity constraints

- Prefer vertical capabilities over disconnected internal models.
- Do not add a package solely to match an architecture diagram.
- Add generic abstractions only after multiple implemented consumers prove shared behavior.
- Keep compatibility serializers as views, never decision authorities.
- Treat configuration order and runtime aliases as execution details, never durable mirror identity.
- Keep ref-policy v1 closed.
- Keep mirrors sequential unless measured need justifies a concurrent execution decision.
- Do not imply cross-remote atomicity or rollback.

## Definition of done

A slice is complete only when observable behavior, failure/recovery tests, versioned contracts, architecture updates, issue acceptance criteria, and CI/security validation are complete.

## Maintenance

Update this plan when a release gate completes, changes order, or is explicitly deferred.
