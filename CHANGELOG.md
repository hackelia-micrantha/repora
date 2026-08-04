# Changelog

Repora records user-visible capability, compatibility, security, and release-process changes here. Internal refactors and test-only changes are omitted unless they change an operator-visible guarantee.

## [Unreleased]

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
## [0.1.0] - YYYY-MM-DD
```

The release manager reviews GitHub-generated notes against this changelog. The changelog is the curated compatibility and operator-impact record; generated notes provide commit and contributor detail.
