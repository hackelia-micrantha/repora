# Continuous integration

Repora exposes checked-in validation commands through `make` so local development and GitHub Actions use the same implementation.

## Portable pull-request validation

Run all portable required checks:

```sh
make check
```

This runs formatting verification, module hygiene, `go vet`, fast race-enabled tests, integration tests, CLI smoke tests, and a native build.

## Granular commands

| Command | Purpose |
| --- | --- |
| `make format-check` | Report Go files that require `gofmt`. |
| `make module-check` | Run `go mod tidy` and fail if `go.mod` or `go.sum` changes. |
| `make vet` | Run `go vet ./...`. |
| `make test` | Run fast tests with `-race -count=1 -short`. |
| `make coverage` | Run the fast test suite with race detection and write coverage profile and summary files under `artifacts/coverage/`. |
| `make integration` | Run integration tests against disposable local Git repositories. |
| `make e2e` | Build the CLI and exercise its command boundary. |
| `make build` | Build the native `repoctl` binary. |
| `make build-all` | Verify all current native and cross-compilation targets. |
| `make workflow-check` | Run `actionlint`, workflow-policy regression tests, and Repora's workflow security policy. |

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

These binaries are unsigned, short-lived, non-release verification artifacts. Successful cross-compilation demonstrates build compatibility only; it does not claim runtime validation or platform support. Releases, signing, provenance attestations, and a formal support matrix require separate work.

## Test classification

Integration tests must call the package's integration guard before performing setup:

```go
requireIntegration(t)
```

The guard skips the test when `go test -short` is active. Adding or renaming an integration test must not require workflow changes.

Integration tests must use `t.TempDir()` or another disposable workspace and must not access real remotes or developer repositories.

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

## GitHub-specific checks

CodeQL, vulnerability scanning, coverage artifacts, and cross-platform CI remain separate from `make check`. They are GitHub-hosted or intentionally isolated because they have different runtime, retention, and dependency characteristics.
