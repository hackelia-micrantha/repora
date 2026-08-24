# Ordered implementation path

Status: Current

## Purpose

This roadmap orders the shortest safe path from the published v0.1 mirror-controller baseline to the next independently verified pre-alpha release. Detailed acceptance criteria live in GitHub issues; `docs/plans/current.md` is the active execution authority.

## Completed baselines

- **Published v0.1:** deterministic default-branch mirror status, exact planning, stale-safe apply, destructive authorization, execution evidence, release archives, and checksums.
- **Current main:** managed README plan/apply, document routing, repository assessments, standalone Nix packaging, GitHub repository/CI posture, documentation posture, hooks posture, bounded commit-history posture, mirror posture, offline policy/reporting, and Bitbucket Cloud mirror transport.
- **Assurance:** explicit unit, integration, contract, CLI E2E, deep/race, static-analysis, security, cross-platform build, Nix, and release-package boundaries.

Merged current-main capability is not published capability until a release is tagged and independently verified.

## Dependency path

### P1 — Reconcile project truth (#139)

Update current documentation and the live backlog so they describe merged behavior, published behavior, the active milestone, and explicit deferrals consistently.

Exit condition: current plan, roadmap, architecture, README, and GitHub issues agree.

### P1 — Representative operator acceptance (#137)

Exercise the integrated read-only and dry-run workflows against representative Hackelia-Micrantha repositories, including the posture convergence path and supported provider topology.

Depends on: #139.

Exit condition: exact-commit operator evidence exists, deterministic contracts reproduce, unavailable evidence remains honest, and release-blocking defects are resolved or tracked.

### P1 — Publish the next pre-alpha baseline (#138)

Select the version, curate the changelog, complete release readiness, publish through the immutable tag workflow, and independently verify downloaded assets.

Depends on: #139 and #137.

Exit condition: the published release contains the declared capability set, all release gates pass, checksums verify independently, and the installed binary reports the exact tag and commit.

### P2 — Maintenance

- refresh or supersede conflicted dependency PR #129 against current `main`;
- decide whether native macOS/Windows runtime smoke coverage has enough release value to justify its operational cost;
- address bounded operator defects discovered by #137.

Maintenance work must not displace the P1 dependency path unless it becomes release-blocking.

## Explicit deferrals

- tags, non-default branches, wildcard refspecs, deleted-ref reconciliation, and concurrent mirror mutation;
- automatic rollback or cross-repository/cross-remote transactions;
- provider provisioning, provider-setting remediation, and automatic pull-request remediation;
- Anthesis runtime transport, authentication, evaluator deployment, and approval workflows;
- arbitrary managed files or automatic managed-artifact mirror propagation;
- GitLab/Bitbucket provider-administration posture adapters without a concrete operator need;
- scanner execution inside posture policy or opaque repository scoring;
- hosted orchestration, package-manager channels, containers, signing, and full provenance attestation.

## Cross-cutting requirements

Every slice must preserve:

- durable identity through `uid` and `provider:path`;
- deterministic output for identical semantic inputs;
- explicit and bounded mutation authority;
- fail-closed destructive and stale-state behavior;
- no credentials in configuration, plans, reports, journals, or diagnostics;
- honest partial failure and unavailable evidence;
- versioned compatibility contracts;
- risk-based test-pyramid and static-analysis coverage;
- documentation changes in the same slice as affected behavior.

## Maintenance rule

Update this document when the active dependency order, milestone, release gate, or explicit deferral changes. Do not use it as a historical completion log; merged pull requests, closed issues, changelog entries, and releases preserve that history.
