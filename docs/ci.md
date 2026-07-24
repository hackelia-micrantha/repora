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
| `make integration` | Run integration tests against disposable local Git repositories. |
| `make e2e` | Build the CLI and exercise its command boundary. |
| `make build` | Build the native `repoctl` binary. |
| `make build-all` | Verify the current cross-compilation targets. |
| `make workflow-check` | Run `actionlint` and Repora's workflow security policy. |

## Test classification

Integration tests must call the package's integration guard before performing setup:

```go
requireIntegration(t)
```

The guard skips the test when `go test -short` is active. Adding or renaming an integration test must not require workflow changes.

Integration tests must use `t.TempDir()` or another disposable workspace and must not access real remotes or developer repositories.

## Workflow policy

Third-party GitHub Actions are pinned to full immutable commit SHAs. A version comment remains beside each SHA so Dependabot updates stay readable.

`make workflow-check` enforces the initial policy:

- workflow syntax and expressions pass `actionlint`;
- third-party actions use a full SHA and readable version comment;
- each workflow declares top-level permissions;
- every job has an explicit timeout;
- `pull_request_target` is prohibited.

Workflows use least-privilege permissions. Untrusted pull-request code must not receive repository secrets or elevated token permissions.

Dependabot checks GitHub Actions and Go modules weekly in separate groups. Minor and patch updates may be grouped; major updates remain separate. Dependency-update pull requests are not auto-merged.

## GitHub-specific checks

CodeQL, vulnerability scanning, and cross-platform CI remain separate from `make check`. They are GitHub-hosted or intentionally isolated because they have different runtime and dependency characteristics.
