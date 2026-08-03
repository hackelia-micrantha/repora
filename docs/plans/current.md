# Current implementation plan

Status: Active

## Release objective

The next meaningful Repora release is a reviewed v0.1 tag of the completed local-first multi-mirror controller.

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
| Multi-mirror mutation | Complete | Sequential independent actions continue after runtime failure without rollback. |
| Apply output and journals | Complete | Apply v3 and execution-record v3 preserve per-target outcomes. |
| Release packaging | Complete in code | Tag-only workflow builds four archives, checksums, metadata, and verification. |
| Release hardening | Active next gate | Reconcile remaining security scope, release checklist, changelog, and bounded benchmark evidence. |

## Immediate sequence

### 1. Validate and merge release packaging

Exit condition:

- pull-request release packaging and verification are green;
- workflow policy confirms tag-only publication and least privilege;
- issue #5 acceptance criteria are reconciled and closed.

### 2. Complete v0.1 hardening

Under #11:

- reconcile remaining #10 security and supply-chain requirements against the existing CI architecture;
- add a concise repeatable release checklist;
- define generated release notes or a manual changelog policy;
- add bounded informational benchmark evidence or explicitly defer it with rationale;
- document reproducibility expectations and limitations;
- run the final tagged-release readiness review.

## Explicit deferrals

- tags, non-default branches, wildcard refspecs, and deleted-ref reconciliation;
- concurrent mirror mutation;
- automatic rollback or cross-repository/cross-remote transactions;
- managed repository artifacts and README templating;
- advanced document routing and repository assessments;
- optional Anthesis policy integration;
- provider provisioning or hosted control-plane behavior;
- package managers, containers, signing, and full provenance.

Deferred tracks must reuse the current plan, policy, execution, result, and evidence substrate rather than create parallel paths.

## Simplicity constraints

- Prefer vertical capabilities over disconnected internal models.
- Keep GitHub Releases as the initial distribution mechanism.
- Keep release packages limited to plain archives and checksums.
- Preserve tag-only publication and least-privilege workflow permissions.
- Treat compatibility serializers as views, never decision authorities.
- Keep ref-policy v1 closed and mirrors sequential.
- Do not imply cross-remote atomicity, rollback, signing, or native validation where only cross-compilation exists.

## Definition of done

A slice is complete only when observable behavior, failure/recovery tests, versioned contracts where applicable, documentation, issue acceptance criteria, independent review, and CI/security validation are complete.

## Maintenance

Update this plan when a release gate completes, changes order, or is explicitly deferred.
