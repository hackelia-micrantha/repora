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
| Release packaging | Complete | Tag-only workflow builds four archives, checksums, metadata, and verification. |
| Security and supply-chain CI | Complete in code | Vulnerability, CodeQL, secret, dependency-license, and workflow-policy validation are defined. |
| Release checklist and changelog policy | Complete in code | Publication, independent verification, failure handling, and curated release notes are documented. |
| Benchmark decision | Explicitly deferred | No stable workload or useful shared-CI threshold exists for v0.1; triggers are documented. |
| First v0.1 publication | Active final gate | Merge the hardening changes, run final readiness review, publish the tag, and verify downloaded assets. |

## Immediate sequence

### 1. Validate and merge v0.1 hardening

Exit condition:

- required CI, security, workflow-policy, and release-package validation are green;
- issue #10 is reconciled against implemented controls;
- issue #11 acceptance criteria are updated to reflect the benchmark deferral and documented release process;
- current architecture, README, release guidance, and this plan agree.

### 2. Publish and independently verify v0.1

Follow `docs/release-checklist.md`.

Exit condition:

- the release tag points to the reviewed `main` commit and is not moved;
- the tag-only workflow publishes four archives plus `checksums.txt`;
- downloaded assets pass checksum verification;
- the Linux binary reports the intended tag and exact commit;
- a bounded local-repository status/plan/dry-run smoke workflow succeeds;
- release evidence is recorded in issue #11 or #27.

### 3. Close the v0.1 milestone

- close #10 after its reconciled security scope is complete;
- close #11 after the published release is independently verified;
- close #27 after the first v0.1 path is validated;
- choose one explicit post-v0.1 milestone before starting deferred product tracks.

## Explicit deferrals

- tags, non-default branches, wildcard refspecs, and deleted-ref reconciliation;
- concurrent mirror mutation;
- automatic rollback or cross-repository/cross-remote transactions;
- managed repository artifacts and README templating;
- advanced document routing and repository assessments;
- Anthesis policy integration;
- provider provisioning or hosted control-plane behavior;
- package managers, containers, signing, and full provenance;
- repository-wide performance gates without a stable workload and threshold.

Deferred tracks must reuse the current plan, policy, execution, result, and evidence substrate rather than create parallel paths. Anthesis is not part of the v0.1 or immediate post-release execution path unless a later explicit decision reprioritizes it.

## Simplicity constraints

- Prefer vertical capabilities over disconnected internal models.
- Keep GitHub Releases as the initial distribution mechanism.
- Keep release packages limited to plain archives and checksums.
- Preserve tag-only publication and least-privilege workflow permissions.
- Treat compatibility serializers as views, never decision authorities.
- Keep ref-policy v1 closed and mirrors sequential.
- Do not imply cross-remote atomicity, rollback, signing, or native validation where only cross-compilation exists.
- Do not add Anthesis integration as release hardening or incidental follow-up.

## Definition of done

A slice is complete only when observable behavior, failure/recovery tests, versioned contracts where applicable, documentation, issue acceptance criteria, independent review, and CI/security validation are complete.

A release is complete only after published assets—not merely local packages or a successful publication workflow—have been independently downloaded and verified.

## Maintenance

Update this plan when a release gate completes, changes order, or is explicitly deferred.
