# Release installation and verification

Status: Current

Repora publishes versioned `repoctl` archives through GitHub Releases. The initial distribution is intentionally limited to plain archives and SHA-256 checksums.

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

Checksums protect download integrity. Initial releases are not cryptographically signed and do not include a full provenance attestation.

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

The command reports both the release tag and exact source commit:

```text
repoctl v0.1.0 (<commit>)
```

## Release construction

A trusted `v*` tag push starts `.github/workflows/release.yml`. The workflow:

1. checks out the tagged source;
2. uses the repository's pinned Go toolchain;
3. derives the source timestamp from the tagged commit;
4. cross-compiles with `CGO_ENABLED=0`, `-trimpath`, and VCS auto-stamping disabled;
5. injects the tag and source commit through linker flags;
6. creates normalized archives containing `repoctl`, `LICENSE`, and `README.md`;
7. generates `checksums.txt`;
8. verifies every checksum, archive member, Linux executable, and embedded version;
9. publishes the files to a GitHub Release only after successful verification.

Pull requests that change the release boundary run the same package and verification scripts with validation metadata, but receive only read permissions and cannot publish a release.

## Local reproduction

From a clean checkout:

```bash
export VERSION=v0.1.0
export COMMIT="$(git rev-parse HEAD)"
export SOURCE_DATE_EPOCH="$(git show -s --format=%ct HEAD)"
make release-package
make release-verify
```

Outputs are written to `dist/`. Re-running with the same source, toolchain, metadata, and platform is expected to produce equivalent archive contents and checksums.

## Rollback

Repora does not include an automatic updater or rollback mechanism. To roll back:

1. download a previously reviewed release;
2. verify its checksum;
3. replace the installed binary;
4. confirm the selected version with `repoctl --version`.

Repository mutation recovery remains separate: after a stale or partial mirror operation, observe current state and create a new exact plan rather than replaying old journal evidence.

## Deferred distribution work

The initial release path does not include Homebrew, Scoop, Nix packages, container images, signing, a hosted update service, or full SLSA provenance. Each requires a separate reviewed issue or decision.