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

## Test classification

Integration tests must call the package's integration guard before performing setup:

```go
requireIntegration(t)
```

The guard skips the test when `go test -short` is active. Adding or renaming an integration test must not require workflow changes.

Integration tests must use `t.TempDir()` or another disposable workspace and must not access real remotes or developer repositories.

## GitHub-specific checks

CodeQL, vulnerability scanning, and cross-platform CI remain separate from `make check`. They are GitHub-hosted or intentionally isolated because they have different runtime and dependency characteristics.
