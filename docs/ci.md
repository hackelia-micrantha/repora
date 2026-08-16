# Continuous integration

Repora exposes checked-in validation commands through `make` so local development and GitHub Actions use the same implementation.

## Portable pull-request validation

Run all portable required checks:

```sh
make check
```

This runs formatting verification, module hygiene, `go vet`, fast race-enabled tests, integration tests, contract tests, CLI smoke tests, and a native build.

`make check` remains portable across Repora's supported Go toolchains. The stronger current-toolchain analyzer is exposed separately through `make static-analysis` and is required by pull-request CI.

## Granular commands

| Command | Purpose |
| --- | --- |
| `make format-check` | Report Go files that require `gofmt`. |
| `make module-check` | Run `go mod tidy` and fail if `go.mod` or `go.sum` changes. |
| `make vet` | Run `go vet ./...`. |
| `make static-analysis` | Run `go vet` and the pinned Staticcheck release against all Go packages and tests. |
| `make test` | Run fast tests with `-race -count=1 -short`. |
| `make coverage` | Run the fast test suite with race detection and write coverage profile and summary files under `artifacts/coverage/`. |
| `make integration` | Run integration-bearing packages against disposable local Git repositories. |
| `make contract-test` | Run routing, context-receipt, and repository-assessment contract validation. |
| `make deep-repeat` | Repeat the fast race-enabled suite, reporting the failing iteration. |
| `make deep-integration` | Run all tests, including integration tests, under the race detector. |
| `make e2e` | Build the CLI and exercise its command boundary. |
| `make build` | Build the native `repoctl` binary. |
| `make build-all` | Verify all current native and cross-compilation targets. |
| `make security-secrets` | Scan the available Git history for secrets using a pinned Gitleaks version with redacted output. |
| `make security-licenses` | Check shipped Go dependencies for forbidden or unknown licenses and write a CSV inventory. |
| `make workflow-check` | Run `actionlint`, workflow-policy regression tests, and Repora's workflow security policy. |
| `make release-package` | Build normalized cross-platform release archives and checksums. |
| `make release-verify` | Verify checksums, archive contents, metadata, and the Linux packaged binary. |

## Static analysis

The pull-request `Lint` job runs `make static-analysis` using the current CI Go toolchain. The target first runs `go vet` and then a pinned Staticcheck release, keeping the analyzer invocation identical between local development and CI.

Staticcheck is intentionally not part of the portable `make check` target because scheduled compatibility validation runs that target on the minimum supported Go toolchain as well as the current toolchain. Analyzer releases have their own Go toolchain support window; keeping the stronger analyzer in a dedicated target prevents minimum-version compatibility from being coupled to that window.

The `mise lint` task delegates Go analysis to `make static-analysis` instead of maintaining a second linter configuration. Workflow and TOML linting remain additional local checks.

Security-oriented static analysis remains independently enforced by CodeQL in the `security` workflow. General static analysis and security analysis are complementary failure boundaries and neither replaces the other.

## Coverage evidence

The unit-test CI job runs `make coverage`. The `go test` output reports coverage for each package, and `go tool cover -func` prints function-level detail and the repository total.

CI retains both `coverage.out` and the text summary for 7 days in the `go-coverage` artifact. Coverage is evidence for review and trend analysis; no repository-wide percentage threshold is enforced.

Coverage profiles contain repository-relative Go source paths and must not be augmented with credentials, environment dumps, or sensitive local paths.

## Verification binaries and target status

CI produces explicitly named binaries and retains each one for 7 days:

| Target | Verification method | Runtime exercised in CI | Formal support status |
| --- | --- | --- | --- |
| Linux amd64 | Built natively on the Linux runner | Yes, through the CLI smoke boundary | No published release-support contract |
| Windows amd64 | Cross-compiled from Linux | No | No published release-support contract |
| macOS amd64 | Cross-compiled from Linux | No | No published release-support contract |
| macOS arm64 | Cross-compiled from Linux | No | No published release-support contract |

Artifact names include the target operating system and architecture. Windows binaries use the `.exe` suffix.

These binaries are unsigned, short-lived, non-release verification artifacts. Successful cross-compilation demonstrates build compatibility only; it does not claim runtime validation or platform support. Release archives use the same target set but pass the stronger package verification documented in [`release.md`](release.md).

## Test classification

The pull-request pipeline keeps distinct validation boundaries rather than treating every test as one undifferentiated suite:

| Layer | CI boundary | Local command | Scope |
| --- | --- | --- | --- |
| Unit / fast | `Unit tests and coverage` | `make test` or `make coverage` | All Go packages with `-short`, race detection, and coverage evidence. |
| Integration | `Integration tests` | `make integration` | Packages whose tests exercise disposable local Git repositories and multi-component behavior. |
| Contract | `Contract tests` | `make contract-test` | Routing, trust, context-receipt, and assessment data/CLI contracts. |
| End-to-end | `CLI end-to-end smoke tests` | `make e2e` | Built `repoctl` process and user-visible command boundary. |

The pyramid should remain bottom-heavy: most behavioral cases belong in fast Go tests; integration tests are reserved for component interactions that require real local Git behavior; end-to-end tests cover only critical CLI paths.

Integration tests must call the package's integration guard before performing setup:

```go
requireIntegration(t)
```

The guard skips the test when `go test -short` is active. Adding or renaming an integration test must not require workflow changes.

Integration tests must use `t.TempDir()` or another disposable workspace and must not access real remotes or developer repositories.

## Scheduled deep validation

`.github/workflows/deep-validation.yml` runs every Sunday at 08:17 UTC and can also be started manually. It is non-blocking for commits that have already merged. Each job is a separate failure boundary:

- **Repeated fast tests** runs the race-enabled short suite ten times. The log records the exact command, package pattern, iteration count, and failing iteration.
- **Full integration race tests** runs every Go test without `-short` under the race detector. Fixtures remain disposable and do not contact provider-hosted remotes.
- **Go compatibility** runs the portable validation contract on Go 1.22.x and the repository's current CI toolchain, Go 1.25.12.

Go 1.22 is the minimum supported toolchain line declared by `go.mod`. Go 1.25.12 is the current validated toolchain. Compatibility jobs are scheduled evidence, not release artifacts.

Reachable-vulnerability scanning, secret detection, license validation, CodeQL, and current-toolchain Staticcheck are intentionally not duplicated in compatibility validation. Their dedicated CI jobs own those checks.

The repository currently has no Go fuzz targets. Bounded fuzzing is deferred until a reviewed `Fuzz...` target exists with a stable corpus boundary. When added, failures must retain the generated corpus input or seed and print a local reproduction command.

### Local reproduction

```sh
make static-analysis
make contract-test
REPEAT_COUNT=10 make deep-repeat
make deep-integration
make check
```

`REPEAT_COUNT` must be a positive integer. The repeated-test runner stops at the first failure and reports the failing iteration.

### Failure ownership and triage

The repository maintainer reviewing automation failures owns initial triage. Scheduled failures are not dismissed as transient without investigation.

1. Record the workflow run, commit SHA, job name, command, and failing iteration or Go version.
2. Reproduce with the documented command using the same commit and toolchain.
3. If reproducible, open a focused issue with logs stripped of credentials and sensitive local paths.
4. If not immediately reproducible, rerun once and file a flake issue containing both results.
5. Fix product or test defects through normal review. Do not weaken timeouts, race coverage, or assertions solely to make the schedule green.

Scheduled workflows use read-only repository permissions, explicit job timeouts, concurrency cancellation, fake Git identities, and no secrets.

## Security validation

`.github/workflows/security.yml` runs on pull requests, pushes to `main`, weekly schedule, and manual dispatch. It contains separate failure boundaries for:

- reachable dependency vulnerabilities through `govulncheck`;
- Git-history secret detection through Gitleaks;
- dependency-license checks and a retained CSV inventory through `go-licenses`;
- CodeQL `security-extended` analysis.

The license inventory is retained for 14 days as `dependency-licenses`. The shipped command package is checked while Repora's own BSL-licensed packages are excluded from third-party dependency classification.

Use a full Git checkout when reproducing `make security-secrets`; a shallow clone cannot prove the same historical scan scope. Scanner logs remain redacted. Do not reproduce a matched secret in an issue or pull request.

Finding thresholds, release blocking, and suppression requirements are defined in [`security-ci.md`](security-ci.md).

## Workflow policy

`make workflow-check` requires Go, network access for the pinned `actionlint` module when it is not cached, and Python 3.10 or newer.

Third-party GitHub Actions are pinned to full immutable commit SHAs. A non-empty readable version comment is mandatory beside each SHA so Dependabot updates remain reviewable.

`make workflow-check` enforces the initial policy:

- workflow syntax and expressions pass `actionlint`;
- third-party actions use a full SHA and readable version comment;
- each workflow declares top-level permissions;
- every job has an explicit timeout;
- `pull_request_target` is prohibited.

Workflows use least-privilege permissions. Untrusted pull-request code must not receive repository secrets or elevated token permissions.

Dependabot checks GitHub Actions and Go modules weekly in separate groups. Minor and patch updates may be grouped; major updates remain separate. Dependency-update pull requests are not auto-merged.

## Release validation

`.github/workflows/release.yml` runs package validation on pull requests that change the release boundary and publishes only from trusted `v*` tag pushes.

Pull-request validation builds the release packages twice and compares checksum manifests before running package verification. The publication job grants `contents: write` only to the tag-only job and rejects a tag commit that is not reachable from `main`.

The complete release process and independent downloaded-asset verification are documented in [`release-checklist.md`](release-checklist.md).

## GitHub-specific checks

CodeQL, vulnerability scanning, secret detection, dependency-license inventory, coverage artifacts, cross-platform CI, scheduled deep validation, and tagged release publication remain separate from `make check`. They are GitHub-hosted or intentionally isolated because they have different runtime, retention, permissions, and dependency characteristics.
