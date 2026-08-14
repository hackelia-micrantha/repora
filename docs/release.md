# Release installation and verification

Status: Current

Repora publishes versioned `repoctl` archives through GitHub Releases. `v0.1.0` established the first published mirror-controller baseline. The archive distribution remains intentionally limited to plain archives and SHA-256 checksums; standalone Nix packaging is a separate repository-owned composition/install surface documented in [`nix.md`](nix.md).

Release operators must follow [`release-checklist.md`](release-checklist.md). User-visible capability, compatibility, security, and release-process changes are curated in [`../CHANGELOG.md`](../CHANGELOG.md).

## Supported release targets

| Operating system | Architecture | Archive |
| --- | --- | --- |
| Linux | amd64 | `repoctl_<version>_linux_amd64.tar.gz` |
| macOS | amd64 | `repoctl_<version>_darwin_amd64.tar.gz` |
| macOS | arm64 | `repoctl_<version>_darwin_arm64.tar.gz` |
| Windows | amd64 | `repoctl_<version>_windows_amd64.zip` |

Linux amd64 packages are executed through the checked-in CLI smoke boundary during release validation. macOS and Windows packages are cross-compiled and archive-validated; cross-compilation alone is not a claim of native runtime testing.

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
install -m 0755 repoctl_<version>_<os>_<arch>/repoctl "$HOME/.local/bin/repoctl"
repoctl --version
```

Windows:

1. Extract the ZIP archive.
2. Move `repoctl.exe` to a directory on `PATH`.
3. Run `repoctl.exe --version`.

The command reports both the release tag and exact source commit, for example:

```text
repoctl v0.1.0 (<commit>)
```

## Release construction

A trusted `v*` tag push starts `.github/workflows/release.yml`. The workflow:

1. checks out the tagged source with full history;
2. refuses publication unless the tag commit is an ancestor of `main`;
3. uses the repository's pinned Go toolchain;
4. derives the source timestamp from the tagged commit;
5. cross-compiles with `CGO_ENABLED=0`, `-trimpath`, and VCS auto-stamping disabled;
6. injects the tag and source commit through linker flags;
7. creates normalized archives containing `repoctl`, `LICENSE`, and `README.md`;
8. generates `checksums.txt`;
9. verifies every checksum, archive member, Linux executable, and embedded version;
10. publishes files to a GitHub Release only after successful verification.

Pull requests that change the release boundary run the same package and verification scripts with validation metadata but receive only read permissions and cannot publish a release. Validation builds the packages twice and requires identical checksum manifests.

Repository administrators should protect release tags so only the intended release process can create `v*` refs. Published version tags must not be moved or reused.

## Release notes and changelog

The release workflow uses GitHub-generated notes for commit/contributor detail. Before creating a tag, the release manager must:

1. move applicable entries from `CHANGELOG.md`'s Unreleased section to `## [<version>] - YYYY-MM-DD`;
2. state which merged capabilities become supported surface in that tag;
3. review operator impact, compatibility, security, and known limitations;
4. compare generated release notes with the curated changelog;
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
2. extract and execute the Linux amd64 binary;
3. confirm `repoctl --version` reports the tag and exact release commit;
4. run a bounded local-repository status, plan, and dry-run smoke workflow;
5. exercise the safest representative path for any newly released CLI capability;
6. record the workflow run, tag, commit, release URL, and verification result in the release issue.

The `v0.1.0` milestone completed this downloaded-asset verification. Every later release repeats the same principle against its own published artifacts.

## Rollback and failed releases

Repora does not include an automatic updater or rollback mechanism. To roll back an installed binary:

1. download a previously reviewed release;
2. verify its checksum;
3. replace the installed binary;
4. confirm the selected version with `repoctl --version`.

If a published release is defective, do not move or reuse its tag. Document the defect, stop recommending the affected version, and publish a reviewed patch version. Preserve failed workflow and verification evidence.

Repository mutation recovery remains separate: after a stale or partial mirror/managed-artifact operation, observe current state and create a new exact plan rather than replaying old journal evidence.

## Nix packaging

The repository now includes a standalone Nix flake for supported Linux/macOS systems. It is validated in pull-request CI and reuses canonical repository checks, but it is not currently published as a separate package-registry channel or substituted for the tagged GitHub archive release process.

See [`nix.md`](nix.md) for build, run, validation, development-shell, and composition guidance.

## Security and benchmark gates

Release security expectations and suppression rules are defined in [`security-ci.md`](security-ci.md). The rationale for not enforcing a repository-wide performance benchmark gate is documented in [`benchmarks.md`](benchmarks.md).

## Deferred distribution work

Current distribution still does not include Homebrew, Scoop, container images, cryptographic release signing, a hosted update service, or full SLSA provenance. Nix is repository-owned standalone packaging rather than a published binary cache/channel. Additional distribution mechanisms require separate reviewed scope.
