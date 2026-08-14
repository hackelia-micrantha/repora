# Standalone Nix packaging

Status: Current

Repora provides a repository-owned Nix flake for standalone installation, validation, development, and composition. The flake is intentionally self-contained: it does not depend on Dubnium modules, checkout paths, credentials, services, or mutation authority.

## Supported systems

The flake currently exports native outputs for:

- `x86_64-linux`;
- `x86_64-darwin`;
- `aarch64-darwin`.

Windows remains supported through Repora's existing cross-platform release archives rather than as a native Nix system.

## Outputs

For each supported Nix system the flake exposes:

```text
packages.<system>.default
packages.<system>.repora
apps.<system>.default
checks.<system>.default
devShells.<system>.default
formatter.<system>
```

The default package and application run the canonical `repoctl` Go CLI. Package and check derivations use the repository's Go 1.25 toolchain contract.

## Build and run

Build the package without installing it globally:

```bash
nix build .#repora
./result/bin/repoctl --version
```

Run the application directly:

```bash
nix run . -- --help
nix run . -- --version
```

The package embeds a bounded development version plus the flake source revision when one is available. Tagged GitHub release archives remain the authoritative versioned distribution channel unless a later release decision changes that boundary.

## Validation

Run all flake checks with:

```bash
nix flake check --print-build-logs
```

The flake does not define a second validation policy. Its checks reuse Repora's canonical repository targets:

| Flake check | Canonical repository boundary |
| --- | --- |
| `format` | `make format-check` |
| `unit` | `make test` |
| `integration` | `make integration` |
| `contract` | `make contract-test` |
| `e2e` | `make e2e` |
| `static-analysis` | `make static-analysis` |
| `smoke` | packaged `repoctl --help` and `repoctl --version` |

The static-analysis check uses Staticcheck `2026.1`, matching the version declared by the Makefile, but consumes the package from the pinned Nixpkgs input so the Nix sandbox does not need to fetch analyzer source at check time.

Repository-mutating integration tests run only against disposable fixtures. The Nix check environment uses an isolated temporary home, disables system Git configuration, and disables interactive credential prompting. Installing or evaluating the package grants no push, pull-request, release, or provider mutation authority.

## Development shell and formatter

Enter the repository development environment with:

```bash
nix develop
```

The shell includes the project-compatible Go toolchain, Git, Make, Python, Staticcheck, and the Nix formatter.

Format the flake with:

```bash
nix fmt
```

Normal repository development commands remain available through `mise` and `make`; Nix is an additional reproducible packaging/composition boundary rather than a replacement build policy.

## Composing Repora from another flake

A consumer can pin Repora as an ordinary flake input and reference the exported package or application without importing any private Micrantha infrastructure:

```nix
{
  inputs.repora.url = "github:hackelia-micrantha/repora";

  outputs = { self, repora, ... }: {
    packages.x86_64-linux.repora = repora.packages.x86_64-linux.repora;
  };
}
```

Consumers may choose their own higher-level service, command, or operator integration. That composition does not transfer Repora's repository-domain logic or mutation decisions into the consuming repository.

Repora keeps its own pinned Nixpkgs input. A consumer may separately test input-following compatibility, but that is not required by the standalone package contract.

## Trust and authority boundary

Nix packaging exposes the CLI and validation outputs only. Runtime Git credentials, repository topology, destructive authorization, exact-plan validation, stale preflight, leases, and execution evidence remain controlled by Repora's existing runtime contracts.

Packaging must therefore never be treated as authorization to mutate a repository merely because the package is installed or composed into another system.
