# Changelog

Repora records user-visible capability, compatibility, security, and release-process changes here. Internal refactors and test-only changes are omitted unless they change an operator-visible guarantee.

## [Unreleased]

## [0.2.0] - 2026-08-24

### Added

- Managed root-README configuration and deterministic non-executable template rendering with contained local template paths.
- Exact `repora.io/managed-artifact-plan` v1 review artifacts, byte-aware diffs, stale-safe dry-run, journaled real apply, isolated candidate commits, and exact reviewed-base leased canonical pushes.
- Managed-artifact apply-result and execution-record v1 contracts with explicit partial-success evidence and no automatic mirror propagation.
- Repository-assessment v1 schemas and local CLI commands for strict report validation, finding projection, scorecard projection, and bounded skeleton creation.
- Deterministic document-routing foundations including trust tiers, context receipts, hierarchical summary-first routing, and bounded Go AST source selectors.
- Read-only GitHub repository/CI posture inventory with versioned observed/unknown/unavailable facts for branch protection, repository hygiene files, workflow permissions/events/runners, and action pinning evidence.
- Deterministic documentation posture with a versioned observation profile, README section/link and exact content-marker facts, routing trust-tier preservation, and no prose scoring or remediation authority.
- Read-only hooks/local-workflow posture with common/custom hook-manager signals, bounded repository-declared expectations, required-check/CI coverage evidence, bootstrap/bypass documentation facts, and static network-load signals without hook execution.
- Read-only bounded commit-history posture with versioned signature, merge-shape, size/file-scope, sensitive-path, and commit/PR-association facts while excluding identity/productivity analytics and unsupported review/direct-push inference.
- Read-only mirror posture with declared canonical/mirror identities, default-branch-name drift, existing reconciliation state/count evidence, bounded provider metadata facts, and explicit unknown tag/release drift under the default-branch-only v1 scope.
- Offline posture policy evaluation with versioned policy-profile, normalized-input, and report contracts; typed adapters consume the completed repository/CI, documentation, hooks, commit-history, and mirror fact domains without re-scanning providers.
- Offline `repoctl posture converge` command that strictly validates captured collector artifacts, rejects duplicate/mixed repository evidence atomically, and emits deterministic normalized policy inputs for `posture report`.
- Deterministic Markdown/JSON posture reporting with explicit expected-versus-observed results, severity, remediation options, warning/failure states, source evidence, time-bounded exceptions, expired-exception findings, and visible unknown/unavailable evidence.
- Bitbucket Cloud mirror transport using provider/path identity and credential-free HTTPS runtime resolution while reusing the existing multi-mirror status/plan/apply pipeline.
- ADR-0018 and the optional additive Anthesis `pre_apply` policy-integration design; runtime evaluator/transport coupling remains deferred.
- Standalone Nix package, app, checks, development shell, and formatter outputs for supported Linux/macOS systems.

### Changed

- CI now exposes explicit fast/unit, integration, contract, CLI end-to-end, and deep/race validation boundaries plus a canonical Staticcheck + `go vet` static-analysis target.
- The current Go 1.25 validation/release toolchain is patched to Go 1.25.13 across local tooling and GitHub workflows.
- Nix validation reuses the same repository build/test/static-analysis targets rather than defining a second quality policy.
- The formatting gate excludes generated vendored dependency source while continuing to check first-party Go files.
- Current documentation now treats managed README, assessment, routing, the published `v0.1.0` baseline, standalone Nix packaging, GitHub posture inventory, documentation posture, hooks/local-workflow posture, bounded commit-history posture, mirror posture, and posture policy/reporting as implemented behavior rather than future work.
- Posture report evaluation requires an explicit `--as-of YYYY-MM-DD` date so exception expiry is deterministic rather than dependent on the executing machine's wall clock.
- Release publication now has a manual fail-closed tag workflow that only creates a new `vMAJOR.MINOR.PATCH` tag at current `main`, refuses existing/stale targets, requires the matching changelog heading, and explicitly dispatches the existing release workflow without moving tag authority into package publication.

### Security

- Managed README mutation uses fixed-path authority, exact current-state preflight, fail-closed INTENT persistence, verified candidate commits, exact base leases, bounded diagnostics, and no force override.
- GitHub posture inventory uses a GET-only provider interface, environment-only optional authentication, explicit unavailable evidence for hidden provider data, bounded workflow normalization, and no mutation-capable provider method.
- Documentation posture reuses the GET-only provider boundary, treats profile/Markdown/router inputs as bounded data, preserves unavailable/unknown evidence, and does not grant policy or mutation authority to repository-owned observation profiles.
- Hooks/local-workflow posture treats repository-owned hook/config/profile/workflow content as bounded data, never installs or executes target-repository hooks, does not follow network-loaded hook references, and keeps CI as the enforcement authority.
- Commit-history posture keeps provider reads GET-only, caps history/profile scope, omits author/committer identity analytics, and preserves unknown instead of inferring direct pushes, missing review, tag signatures, or release boundaries without proof.
- Mirror posture reuses the existing fetch-only local reconciliation observation path, never calls push/synchronization/provider-mutation operations, and preserves unavailable provider metadata instead of inferring healthy or drifted state.
- Posture convergence/policy/reporting is offline-only, strictly parses versioned collector/policy/fact inputs, preserves evidence gaps, rejects cross-repository fact mixing and partial adapter mutation, and does not grant provider access, execute repository configuration, run scanners, or create an opaque numeric score.
- Bitbucket mirrors require exact `workspace/repository` provider paths, reject legacy/credential-bearing transport input, keep credentials external to configuration/evidence, and leave unsupported provider-administration posture metadata explicit `unavailable`.
- Routing and assessment commands remain read-only with respect to Git/provider state unless a separately reviewed mutation boundary explicitly applies.
- CI preserves immutable action pins, least-privilege workflow permissions, reachable-vulnerability scanning, CodeQL, full-history secret detection, and dependency-license validation.

## [0.1.0] - 2026-08-03

### Added

- Local-first multi-mirror Git controller for one GitLab canonical and one or more GitHub/GitLab mirrors.
- Stable `uid` repository identity and provider/path-bound mirror identity.
- Independent multi-mirror status with `EQUAL`, `BEHIND`, `AHEAD`, `DIVERGED`, and `ERROR` outcomes.
- Closed default-branch-only reference policy with explicit destructive-change authorization.
- Deterministic reconciliation artifact v2 and versioned CLI/status/apply schemas.
- Complete all-target stale-ref preflight and force-with-lease protection.
- Sequential independent mirror execution with honest partial outcomes and no rollback claim.
- Immutable execution-record v3 intent/result evidence.
- Cross-platform release archives for Linux amd64, macOS amd64/arm64, and Windows amd64 with SHA-256 checksums.

### Security

- Credential-bearing URL rejection and diagnostic redaction.
- Symlink-safe local cache and journal boundaries.
- Race-enabled, failure-path, workflow-policy, vulnerability, CodeQL, secret, and dependency-license validation.

### Known limitations

- GitLab canonical repositories only.
- Default branch only; no tags, wildcard refs, or deleted-ref reconciliation.
- No provider provisioning, automatic rollback, cross-remote transaction, signing, or full provenance attestation.
- macOS and Windows archives are cross-compiled but not natively exercised in CI.

## Release process

Before publishing a version, move the applicable Unreleased entries under a version heading in the form:

```text
## [<version>] - YYYY-MM-DD
```

The release manager reviews GitHub-generated notes against this changelog. The changelog is the curated compatibility and operator-impact record; generated notes provide commit and contributor detail.
