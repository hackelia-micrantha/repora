# Release installation and verification

Status: Current

Repora publishes versioned `repoctl` archives through GitHub Releases. `v0.1.0` established the first published mirror-controller baseline. Release archives are complete public Unix-style distributions: they contain the executable, man page, safe example configuration, public JSON schemas, license, and README. SHA-256 checksums cover every published archive. Standalone Nix packaging exposes the same public runtime/support surface as a separate repository-owned composition/install path documented in [`nix.md`](nix.md).

Release operators must follow [`release-checklist.md`](release-checklist.md). User-visible capability, compatibility, security, and release-process changes are curated in [`../CHANGELOG.md`](../CHANGELOG.md).

## Supported release targets

| Operating system | Architecture | Archive |
| --- | --- | --- |
| Linux | amd64 | `repoctl_<version>_linux_amd64.tar.gz` |
| macOS | amd64 | `repoctl_<version>_darwin_amd64.tar.gz` |
| macOS | arm64 | `repoctl_<version>_darwin_arm64.tar.gz` |
| Windows | amd64 | `repoctl_<version>_windows_amd64.zip` |

Linux amd64 packages are executed through the checked-in CLI smoke boundary during release validation. macOS and Windows packages are cross-compiled and archive-validated; cross-compilation alone is not a claim of native runtime testing.

Each archive has the same support layout beneath its package root:

```text
repoctl                         # repoctl.exe on Windows
LICENSE
README.md
share/man/man1/repoctl.1
share/repora/examples/repora.yaml
share/repora/schemas/*.schema.json
```

The example configuration is intentionally non-secret. Runtime credentials and real operator topology are not release artifacts.

## Download and verify

Download the archive for the required target and `checksums.txt` from the same GitHub Release.

On Linux:

```bash
sha256sum -c checksums.txt --ignore-missing
```

On macOS:

```bash
shasum -a 256 repoctl_<version>_darwin_arm64.tar.gz
# Compare the result with the matching line in checksums.txt.
```

On Windows PowerShell:

```powershell
Get-FileHash .\repoctl_<version>_windows_amd64.zip -Algorithm SHA256
# Compare the result with the matching line in checksums.txt.
```

Checksums protect download integrity. Releases are not currently cryptographically signed and do not include a full provenance attestation.

## Install

Linux or macOS:

```bash
tar -xzf repoctl_<version>_<os>_<arch>.tar.gz
package="repoctl_<version>_<os>_<arch>"
mkdir -p "$HOME/.local/bin" "$HOME/.local/share/man/man1" "$HOME/.local/share/repora/examples" "$HOME/.local/share/repora/schemas"
install -m 0755 "$package/repoctl" "$HOME/.local/bin/repoctl"
install -m 0644 "$package/share/man/man1/repoctl.1" "$HOME/.local/share/man/man1/repoctl.1"
install -m 0644 "$package/share/repora/examples/repora.yaml" "$HOME/.local/share/repora/examples/repora.yaml"
cp "$package"/share/repora/schemas/*.schema.json "$HOME/.local/share/repora/schemas/"
repoctl --version
```

If `$HOME/.local/share/man` is not already in the system manpath, use `man -l "$HOME/.local/share/man/man1/repoctl.1"` or configure the local manpath explicitly.

Windows:

1. Extract the ZIP archive.
2. Move `repoctl.exe` to a directory on `PATH`.
3. Keep or copy the `share/repora` reference files wherever local tooling expects documentation/schema assets.
4. Run `repoctl.exe --version`.

The command reports both the release tag and exact source commit, for example:

```text
repoctl v0.2.1 (<commit>)
```

## Configuration ownership

Repora release artifacts ship only a safe example `repora.yaml`. The real operator configuration remains owned by the consuming host or configuration repository. Current mirror commands default to a `repora.yaml` in the working directory and also accept an explicit `-f` path.

Do not embed provider credentials, tokens, private keys, or sensitive host-specific state into release artifacts or Nix derivations. Git/SSH credential helpers, environment-scoped provider tokens, and other runtime credential mechanisms remain separate authority boundaries.

A NixOS/Home Manager consumer should therefore pin the Repora release flake and independently manage the actual `repora.yaml` it wants to pass to the CLI. The public Repora repository owns distribution; the consuming host owns deployment and runtime authority.

## Release construction

The preferred release action is the manually dispatched `.github/workflows/release-tag.yml` workflow. It accepts a new `vMAJOR.MINOR.PATCH` value and an optional exact commit SHA, then fails closed unless:

- the requested version is a plain semantic-version tag;
- an explicitly supplied commit is a full exact commit SHA;
- the target commit is reachable from current `origin/main`;
- the target commit's own `CHANGELOG.md` contains a dated heading for that version; and
- the tag does not already exist.

If the commit input is omitted, the workflow uses current `main`. Supplying the exact reviewed release commit is preferred because `main` may advance after release validation without invalidating an already-reviewed ancestor.

The workflow creates a lightweight immutable tag without force and explicitly dispatches `.github/workflows/release.yml` with that existing tag. The explicit dispatch is required because events generated with the repository `GITHUB_TOKEN` do not normally trigger another workflow from a resulting tag push. No PAT or additional release secret is required.

An externally created trusted `v*` tag push remains supported and also starts `.github/workflows/release.yml`. The release workflow can therefore publish either from a normal tag-push event or from the explicit dispatch created by the tag workflow. In both cases it:

1. checks out the tagged source with full history;
2. validates the semantic-version tag and refuses publication unless the checked-out tag commit is an ancestor of `main`;
3. uses the repository's pinned Go toolchain;
4. derives the source timestamp from the tagged commit;
5. cross-compiles with `CGO_ENABLED=0`, `-trimpath`, and VCS auto-stamping disabled;
6. injects the tag and exact tagged source commit through linker flags;
7. creates normalized archives containing `repoctl`, `LICENSE`, `README.md`, `repoctl(1)`, the safe example configuration, and all checked-in public `*.schema.json` contracts;
8. generates `checksums.txt`;
9. verifies every checksum, required archive member, public schema member, Linux executable, and embedded version; and
10. publishes files to a GitHub Release only after successful verification.

A manual dispatch of `release.yml` with no `publish_tag` remains validation-only. Supplying `publish_tag` is publication-capable only for an already-existing immutable tag whose checked-out source satisfies the same main-ancestry and package-verification checks.

Pull requests that change the release boundary run the same package and verification scripts with validation metadata but receive only read permissions and cannot publish a release. Validation builds the packages twice and requires identical checksum manifests. Changes to the packaged man page, example configuration, or public schemas are themselves release-boundary changes and trigger that validation.

Repository administrators should protect release tags so only the intended release process can create `v*` refs. Published version tags must not be moved or reused.

## Release notes and changelog

The release workflow uses GitHub-generated notes for commit/contributor detail. Before creating a tag, the release manager must:

1. move applicable entries from `CHANGELOG.md`'s Unreleased section to `## [<version>] - YYYY-MM-DD`;
2. state which merged capabilities become supported surface in that tag;
3. review operator impact, compatibility, security, and known limitations;
4. compare generated release notes with the curated changelog; and
5. add missing upgrade, limitation, or security context to the published release description.

Generated notes are not compatibility authority. The changelog is the curated user-facing record, and a feature being merged on `main` is not by itself a release decision.

## Local reproduction

The checked-in archive scripts currently target a Linux build environment with Bash, GNU `tar`, GNU `touch`, `gzip`, `zip`, `unzip`, and `sha256sum`.

From a clean checkout in that environment:

```bash
export VERSION=vX.Y.Z
export COMMIT="$(git rev-parse HEAD)"
export SOURCE_DATE_EPOCH="$(git show -s --format=%ct HEAD)"
make release-package
make release-verify
```

Outputs are written to `dist/`. Re-running with the same source, Go toolchain, metadata, and packaging tools is expected to produce identical archive checksums. The pull-request workflow verifies that expectation by comparing two complete builds.

## Independent post-publication verification

A successful publication job is necessary but not sufficient. After publication, download the release assets from GitHub and verify them independently:

1. verify each archive against the published `checksums.txt`;
2. confirm the man page, safe example configuration, and public schemas are present in the downloaded archive;
3. extract and execute the Linux amd64 binary;
4. confirm `repoctl --version` reports the tag and exact release commit;
5. run a bounded local-repository status, plan, and dry-run smoke workflow;
6. exercise the safest representative path for any newly released CLI capability; and
7. record the workflow run, tag, commit, release URL, and verification result in the release issue.

The `v0.1.0` milestone completed this downloaded-asset verification. Every later release repeats the same principle against its own published artifacts.

## Rollback and failed releases

Repora does not include an automatic updater or rollback mechanism. To roll back an installed binary:

1. download a previously reviewed release;
2. verify its checksum;
3. replace the installed binary and matching support files; and
4. confirm the selected version with `repoctl --version`.

If a published release is defective, do not move or reuse its tag. Document the defect, stop recommending the affected version, and publish a reviewed patch version. Preserve failed workflow and verification evidence.

Repository mutation recovery remains separate: after a stale or partial mirror/managed-artifact operation, observe current state and create a new exact plan rather than replaying old journal evidence.

## Nix packaging

The repository includes a standalone Nix flake for supported Linux/macOS systems. The flake installs the same public support contract as the archive distribution: `repoctl`, `repoctl(1)`, the safe example configuration, and the public JSON schemas. Its smoke check fails if those support files are missing.

Downstream systems should consume an immutable release tag or exact revision rather than floating `main`. Repora keeps its own pinned Nixpkgs input; the downstream host remains responsible for selecting the package, providing the actual operator configuration, and supplying runtime credentials.

See [`nix.md`](nix.md) for build, run, validation, development-shell, and composition guidance.

## Security and benchmark gates

Release security expectations and suppression rules are defined in [`security-ci.md`](security-ci.md). The rationale for not enforcing a repository-wide performance benchmark gate is documented in [`benchmarks.md`](benchmarks.md).

Source availability and the use of stripped Go release binaries are not security boundaries. Public release consumers can inspect source, binaries, schemas, and behavior. Secrets and privileged decisions must therefore remain outside distributed artifacts and be enforced through explicit credentials, policy, validation, and runtime controls.

## Deferred distribution work

Current distribution still does not include Homebrew, Scoop, container images, cryptographic release signing, a hosted update service, or full SLSA provenance. Nix is repository-owned standalone packaging rather than a published binary cache/channel. Additional distribution mechanisms require separate reviewed scope.
