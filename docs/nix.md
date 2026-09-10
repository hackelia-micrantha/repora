# Standalone Nix packaging

Status: Current

Repora provides a repository-owned Nix flake for standalone installation, validation, development, and composition. The flake is intentionally self-contained: it does not depend on Dubnium modules, checkout paths, credentials, services, or mutation authority.

## Supported systems

The flake currently exports native outputs for:

- `x86_64-linux`;
- `x86_64-darwin`;
- `aarch64-darwin`.

Windows remains supported through Repora's existing cross-platform release archives rather than as a native Nix system.

## Outputs and installed surface

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

The package is a complete public Unix distribution surface rather than a bare executable. A successful build contains:

```text
bin/repoctl
share/man/man1/repoctl.1
share/repora/examples/repora.yaml
share/repora/schemas/*.schema.json
```

The installed example configuration and schemas are public reference material only. They do not carry credentials, host-specific repository inventory, provider mutation authority, or local policy decisions.

## Build and run

Build the package without installing it globally:

```bash
nix build .#repora
./result/bin/repoctl --version
man -l ./result/share/man/man1/repoctl.1
```

Inspect the packaged safe configuration example and schemas with:

```bash
cat ./result/share/repora/examples/repora.yaml
ls ./result/share/repora/schemas
```

Run the application directly:

```bash
nix run . -- --help
nix run . -- --version
```

The package embeds the version declared by the flake plus the flake source revision when one is available. Release-preparation commits align that package version with the immutable release tag so a consumer pinned to a tagged source gets the same `vMAJOR.MINOR.PATCH` CLI version identity as the release archives. Tagged GitHub release archives remain the authoritative prebuilt distribution channel, while the tagged flake is the authoritative Nix composition source.

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
| `smoke` | packaged `repoctl --help`, `repoctl --version`, man page, safe example config, and public schema presence |

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

A consumer can pin Repora as an ordinary flake input and reference the exported package or application without importing any private Micrantha infrastructure. Long-lived consumers should pin an immutable release tag or exact revision rather than floating on `main`:

```nix
{
  inputs.repora.url = "github:hackelia-micrantha/repora/v0.2.1";

  outputs = { self, repora, ... }: {
    packages.x86_64-linux.repora = repora.packages.x86_64-linux.repora;
  };
}
```

A Home Manager consumer can install the tagged package while keeping its operator configuration in the consuming configuration repository:

```nix
{ pkgs, repora, ... }:
{
  home.packages = [
    repora.packages.${pkgs.stdenv.hostPlatform.system}.repora
  ];

  xdg.configFile."repora/repora.yaml".source = ./repora.yaml;
}
```

Repora does not implicitly read `~/.config/repora/repora.yaml`; current mirror commands default to `./repora.yaml` or an explicit `-f` path. The example above intentionally separates package composition from host-owned configuration so callers can pass the desired path explicitly.

Consumers may choose their own higher-level service, command, or operator integration. That composition does not transfer Repora's repository-domain logic or mutation decisions into the consuming repository.

Repora keeps its own pinned Nixpkgs input. A consumer may separately test input-following compatibility, but that is not required by the standalone package contract.

## Trust and authority boundary

Nix packaging exposes the CLI, man page, safe example configuration, public schemas, and validation outputs only. Runtime Git credentials, real repository topology, destructive authorization, exact-plan validation, stale preflight, leases, and execution evidence remain controlled by Repora's existing runtime contracts.

Do not put credentials, tokens, private keys, or sensitive host-specific values into `flake.nix`, Nix option values that materialize files, or other derivation inputs. Derivation inputs and resulting store paths are not a secret-storage boundary.

Packaging must therefore never be treated as authorization to mutate a repository merely because the package is installed or composed into another system. Source visibility, stripped binaries, and resistance to disassembly are likewise not security boundaries; privileged authority must remain outside distributed artifacts.
