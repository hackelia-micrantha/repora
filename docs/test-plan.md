# Repository TestPlan pilot

The repository-owned `testplan.yaml` declares Repora's **intended required and optional validation dimensions** for Micrantha's Testule Pilot 2 ([tracking issue #178](https://github.com/hackelia-micrantha/repora/issues/178), [organization rollout #36](https://github.com/hackelia-micrantha/hackelia-micrantha/issues/36)). This is a **declaration-only first slice**. Structural validation, a green Go workflow, or the presence of an example test **does not** establish that a Testule requirement is satisfied. No revision-bound Evidence importer or complete `testule gaps` gate is installed in Repora yet.

## Native validation remains authoritative

`Makefile`, `docs/ci.md`, `.github/workflows/ci.yml`, the separate security workflow, and the release checklist define current validation. Preserve the distinct native fast/race, integration, routing/receipt/assessment contract, built-CLI E2E, workflow policy, CodeQL, dependency, license, release packaging, and Nix package checks. Repora's Go 1.25.13 CI and minimum compatibility line do not change merely because Testule's source currently requires Go 1.26.0. The first executable consumer slice must select a pinned, reviewable toolchain or verified artifact without requiring default PR jobs to obtain private GitHub credentials.

## Evidence candidates to review, not passing Evidence

| Plan requirement | Current candidate or native boundary | Review needed before claiming satisfaction |
| --- | --- | --- |
| `level.unit`, `behavior.positive`, `generation.example` | `internal/plan.TestReconcileIsDeterministicAndDoesNotMutateInputs` | Verify the exact assertions and native JSON event on the selected revision. |
| `level.component` | `internal/apply.TestPreflightRepositoryArtifactAuditedChecksEveryTargetBeforeMutation` | Confirm the assertion covers multi-target preflight and side-effect ordering, not whole-repository integration. |
| `level.contract` | `cmd/repoctl.TestStatusOutputMatchesGoldenContract` and `make contract-test` | Keep the specific Go JSON target separate from the Python/Go script-based routing and receipt contracts. |
| `level.integration` | `internal/apply_test.TestExecuteSynchronizesBehindMirrorUsingLocalGitRepos` (native Go package `repoctl/internal/apply`) | Confirm the real disposable-local-Git setup and exact observed package/target; this target is not a remote-hosted provider test. |
| `level.endToEnd` | `make e2e` → `scripts/ci/cli-smoke.sh` on the **built** `repoctl` | This is currently a shell process boundary, **not a Go `test -json` target**. Do not label a unit test E2E merely to fill the row. A bounded generic process/contract Evidence source or dedicated reviewed native target is needed. |
| `behavior.negative` | `internal/plan.TestReconcileRejectsUnsupportedTopologyWithoutPartialPlan` | Verify the rejected topology and absence of a partial mutation plan. |
| `behavior.boundary` | `internal/apply.TestPreflightRepositoryArtifactAuditedRejectsUnknownTargetBeforeGitReads` | Review exact target-identity and no-Git-read assertions; choose a distinct size/time/path boundary target if this is insufficient. |
| `behavior.adversarial` | `internal/git.TestEnsureMirrorRejectsSymlinkEscape` | Confirm real path/symlink escape rejection; do not infer protection against every adversarial class from this one fixture. |

The importer must bind the **exact checked-out Git revision** and canonical plan fingerprint, verify target-level and package-level success from bounded native `go test -json` output, and use explicit reviewed semantic annotations. The imported record must not claim Testule executed tests that native Go actually ran. A deliberately incomplete representative mapping must still return actual Testule exit 5 before a complete mapping may become a blocking CI gate.

## Generation and deployment disposition

Example-based validation is required. Generated, property-based, **fuzz**, and model-based generation are optional in this first plan. `docs/ci.md` currently records **no Go fuzz target**: optional fuzz is an honest visible gap, not an inapplicability decision or completed evidence. Revisit its priority with an actual stable corpus and native bounded fuzz campaign. AI-assisted test generation is not an authoritative correctness requirement; independently verified AI-generated fixtures may still be useful. There is no currently deployed service/system boundary distinct from Repora's local integration and built-CLI E2E boundaries.

## Artifact and release evidence are separate

PR Linux verification binaries execute the CLI smoke; Windows/macOS binaries are cross-compiled but not run on those platforms. Nix package smoke checks its executable and installed support files. These observations **do not** verify a published release asset. Release candidate and post-publication evidence require exact commit/tag/asset identity, digest, archive contents, checksum verification, packaged-binary execution, and supported installation path as specified by `docs/release-checklist.md`. Do not satisfy a TestPlan row by relabeling an unrelated source-tree or cross-compilation check.

## Acceptance before enabling a Testule gate

1. Validate this YAML with an actual pinned Testule `v1alpha1` validator; keep this slice draft until the validation and toolchain source are documented.
2. Review the exact assertions and native events for each candidate. Record missing dimensions rather than attaching invented annotations.
3. Resolve the Go 1.25.13 / Testule Go 1.26 toolchain and distribution decision without silently raising Repora's supported Go version or adding an untrusted artifact source.
4. Preserve native correctness checks and run a representative intentionally incomplete Evidence smoke before a complete, exact-revision gap gate.
5. Qualify Nix/build outputs and released assets separately; update #178 and organization rollout #36 only with actual validation and merge evidence.
