# Release checklist

Status: Current

Use this checklist for Repora GitHub Releases until the distribution model changes. `v0.1.0` established the released mirror-controller baseline; later releases must explicitly identify which post-v0.1 capabilities are included. The current publication mechanism remains GitHub Releases with plain archives and SHA-256 checksums.

## 1. Scope and authority

- [ ] The release commit is on `main`.
- [ ] The intended tag is a new immutable semantic version matching `v*`.
- [ ] The release scope is stated explicitly, including which Unreleased capabilities become supported release surface.
- [ ] No unresolved P0 issue exists.
- [ ] Any unresolved P1 issue is explicitly accepted as non-blocking with rationale in the release issue.
- [ ] Current docs do not claim deferred capabilities are released or production-ready.
- [ ] Expanded ref scope, provider mutation, Anthesis runtime coupling, signing/provenance, hosted services, or other new authority require their own accepted implementation/release decision before inclusion.

A capability being merged on `main` is not by itself a release decision. The release issue/changelog define what the tagged version promises.

## 2. Validation and security

- [ ] Required CI is green on the exact release commit.
- [ ] The scheduled or manually dispatched deep-validation workflow is green on the release commit or has a documented equivalent run.
- [ ] `govulncheck` reports no reachable known vulnerability.
- [ ] CodeQL has no unresolved release-blocking high-confidence finding.
- [ ] Gitleaks reports no unreviewed secret finding.
- [ ] Dependency-license validation passes and the retained license inventory has been reviewed.
- [ ] Static analysis (`go vet` + pinned Staticcheck contract) passes.
- [ ] Applicable unit, integration, contract, and CLI E2E boundaries pass.
- [ ] Workflow policy confirms immutable action pins, explicit permissions, timeouts, and no unsafe `pull_request_target` execution.
- [ ] Release publication retains job-scoped `contents: write`; pull-request validation remains read-only.
- [ ] If Nix packaging changed, `nix flake check` and the packaged app smoke boundary pass on the reviewed revision.

Security suppressions require a repository-visible explanation identifying the tool, finding, evidence, scope, owner, and review trigger. Do not suppress a finding only to make CI green.

## 3. Compatibility and documentation

- [ ] `README.md`, `docs/architecture/current-system.md`, and `docs/plans/current.md` agree on current capability and limitations.
- [ ] Versioned schemas and migration documents match emitted contracts.
- [ ] `CHANGELOG.md` moves applicable Unreleased entries to `## [<version>] - YYYY-MM-DD`.
- [ ] Generated GitHub release notes are reviewed against the curated changelog.
- [ ] Installation, checksum verification, local reproduction, and rollback guidance remain accurate.
- [ ] Known limitations clearly state that checksums provide integrity, not publisher authentication.
- [ ] Cross-compiled targets are not described as natively tested.
- [ ] Managed-artifact, assessment, routing, Nix, or policy-design documentation is included/reconciled when that surface is part of the release scope.

## 4. Package reproduction

From a clean checkout of the release commit:

```bash
export VERSION=vX.Y.Z
export COMMIT="$(git rev-parse HEAD)"
export SOURCE_DATE_EPOCH="$(git show -s --format=%ct HEAD)"
make release-package
cp dist/checksums.txt /tmp/repora-checksums.first.txt
make release-package
diff -u /tmp/repora-checksums.first.txt dist/checksums.txt
make release-verify
```

- [ ] Repeated packaging produces an identical checksum manifest.
- [ ] Every archive contains only the expected binary, `README.md`, and `LICENSE`.
- [ ] The Linux packaged binary passes the CLI smoke boundary.
- [ ] The embedded version and commit match the intended tag and release commit.

## 5. Publication

- [ ] Create the protected tag from the reviewed release commit.
- [ ] Confirm the tag-triggered release workflow starts.
- [ ] Confirm the workflow rejects tags not reachable from `main`.
- [ ] Confirm all package verification completes before publication.
- [ ] Confirm the GitHub Release contains four archives and `checksums.txt` unless a reviewed distribution decision changes the target set.
- [ ] Do not move or reuse a published version tag.

## 6. Independent post-publication verification

Download the published assets rather than using local `dist/` files.

- [ ] Verify every downloaded asset against `checksums.txt`.
- [ ] Extract and execute the Linux amd64 binary.
- [ ] Confirm `repoctl --version` reports the published tag and exact commit.
- [ ] Run a bounded local status/plan/dry-run smoke workflow for the mirror-controller baseline.
- [ ] Exercise any newly released CLI surface with its safest representative read/dry-run path.
- [ ] Record the release URL, workflow run, tag, commit, and verification result in the release issue.

## 7. Failure handling

If publication is wrong or verification fails:

- do not move the tag;
- stop recommending the affected release;
- mark the release as a prerelease or document the defect prominently when appropriate;
- fix through a new reviewed commit and patch version;
- preserve failed workflow and verification evidence;
- never replay an old Repora reconciliation or managed-artifact plan as part of release recovery.

## Exit condition

A release is complete only after the published assets—not merely the workflow or local packages—have passed checksum, metadata, extraction, version, and Linux smoke verification, plus any explicitly declared verification needed for newly released capability.
