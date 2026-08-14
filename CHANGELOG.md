# Changelog

Repora records user-visible capability, compatibility, security, and release-process changes here. Internal refactors and test-only changes are omitted unless they change an operator-visible guarantee.

## [Unreleased]

### Added

- Managed root-README configuration and deterministic non-executable template rendering with contained local template paths.
- Exact `repora.io/managed-artifact-plan` v1 review artifacts, byte-aware diffs, stale-safe dry-run, journaled real apply, isolated candidate commits, and exact reviewed-base leased canonical pushes.
- Managed-artifact apply-result and execution-record v1 contracts with explicit partial-success evidence and no automatic mirror propagation.
- Repository-assessment v1 schemas and local CLI commands for strict report validation, finding projection, scorecard projection, and bounded skeleton creation.
- Deterministic document-routing foundations including trust tiers, context receipts, hierarchical summary-first routing, and bounded Go AST source selectors.
- ADR-0018 and the optional additive Anthesis `pre_apply` policy-integration design; runtime evaluator/transport coupling remains deferred.
- Standalone Nix package, app, checks, development shell, and formatter outputs for supported Linux/macOS systems.

### Changed

- CI now exposes explicit fast/unit, integration, contract, CLI end-to-end, and deep/race validation boundaries plus a canonical Staticcheck + `go vet` static-analysis target.
- Nix validation reuses the same repository build/test/static-analysis targets rather than defining a second quality policy.
- The formatting gate excludes generated vendored dependency source while continuing to check first-party Go files.
- Current documentation now treats managed README, assessment, routing, the published `v0.1.0` baseline, and standalone Nix packaging as implemented behavior rather than future work.

### Security

- Managed README mutation uses fixed-path authority, exact current-state preflight, fail-closed INTENT persistence, verified candidate commits, exact base leases, bounded diagnostics, and no force override.
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
