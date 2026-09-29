# Repository TestPlan pilot

The repository-owned `testplan.yaml` declares Repora's **intended required and optional validation dimensions** for Micrantha's Testule Pilot 2 ([tracking issue #178](https://github.com/hackelia-micrantha/repora/issues/178), [organization rollout #36](https://github.com/hackelia-micrantha/hackelia-micrantha/issues/36)). The declaration slice is merged and semantically validated, but structural validation, a green Go workflow, or the presence of an example test **does not** establish that a Testule requirement is satisfied. Repora now stages one bounded exact native observation source for later import; no normalized Repora Testule Evidence or complete `testule gaps` gate is installed yet.

## Native validation remains authoritative

`Makefile`, `docs/ci.md`, `.github/workflows/ci.yml`, the separate security workflow, and the release checklist define current validation. Preserve the distinct native fast/race, integration, routing/receipt/assessment contract, built-CLI E2E, workflow policy, CodeQL, dependency, license, release packaging, and Nix package checks. Repora's Go 1.25.13 CI and minimum compatibility line do not change merely because Testule's source currently requires Go 1.26.0. The first executable consumer slice must select a pinned, reviewable toolchain or verified artifact without requiring default PR jobs to obtain private GitHub credentials.

## Evidence candidates to review, not passing Evidence

| Plan requirement | Current candidate or native boundary | Review needed before claiming satisfaction |
| --- | --- | --- |
| `level.unit`, `behavior.positive`, `generation.example` | `internal/plan.TestReconcileIsDeterministicAndDoesNotMutateInputs` | CI stages and verifies the exact native JSON event and revision for this target. That source still requires explicit Testule import/annotation before any row is satisfied. |
| `level.component` | `internal/apply.TestPreflightRepositoryArtifactAuditedChecksEveryTargetBeforeMutation` | Reviewed candidate: stale second-target preflight is detected before mutation and journal outcomes retain skipped/stale state. |
| `level.contract` | `cmd/repoctl.TestStatusOutputMatchesGoldenContract` | Reviewed candidate: marshalled status JSON must match the committed golden contract byte-for-byte. Broader Python/Go routing/receipt contracts remain separate native checks. |
| `level.integration` | `repoctl/internal/apply.TestExecuteSynchronizesBehindMirrorUsingLocalGitRepos` | Reviewed candidate: non-short execution uses disposable real local Git repositories and proves BEHIND → apply → EQUAL. A skipped observation is non-evidence. |
| `level.endToEnd` | `make e2e` → `scripts/ci/cli-smoke.sh` on the **built** `repoctl` | This is currently a shell process boundary, **not a Go `test -json` target**. Do not label a unit test E2E merely to fill the row. A bounded generic process/contract Evidence source or dedicated reviewed native target is needed. |
| `behavior.negative` | `repoctl/internal/plan.TestReconcileRejectsUnsupportedTopologyWithoutPartialPlan` | Reviewed candidate: unsupported topologies return repository-specific errors and no partial actions. |
| `behavior.boundary` | `repoctl/cmd/repoctl.TestApplyHonorsParallelLimit` | Reviewed narrow boundary candidate: configured apply parallelism of 1 is enforced; this does not claim all size/time/path/resource boundaries. |
| `behavior.adversarial` | `repoctl/internal/git.TestEnsureMirrorRejectsSymlinkEscape` | Reviewed candidate: a real symlink escape is rejected and no repository is materialized outside the workspace. |

The importer must bind the **exact checked-out Git revision** and canonical plan fingerprint, verify target-level and package-level success from bounded native `go test -json` output, and use explicit reviewed semantic annotations. The imported record must not claim Testule executed tests that native Go actually ran. A deliberately incomplete representative mapping must still return actual Testule exit 5 before a complete mapping may become a blocking CI gate.

## Bounded representative native observation source

`make testule-observation-source` runs only the reviewed planner target with `go test -count=1 -json`. The producer stream is drained to EOF through a one-MiB exclusive mode-0600 capture. Overflow fails closed after draining, so a retained prefix is never trusted and cannot turn producer SIGPIPE into a misleading native result. A separate parser requires the exact package/target to have a `run` event and exactly one terminal `pass`; missing, skipped, failed, malformed, or mismatched observations fail the source step.

Successful PR/main CI retains:

- `artifacts/testule/representative-go-test.json` — bounded native Go event stream;
- `artifacts/testule/subject-revision.txt` — exact checked-out Git commit. On `pull_request`, this is GitHub's tested synthetic merge revision; on `main`, it is the landed commit. The later importer must bind to this tested subject, not relabel it as the PR branch head.

These files are **native observation source material**, not Testule Evidence. They do not contain a Testule plan fingerprint, do not assign Testule semantics by themselves, and do not make the repository plan complete. The later consumer slice must use a qualified Testule executable, bind the canonical plan fingerprint and revision, import the explicit reviewed `unit/positive/example` annotation, and prove that this deliberately representative-only input still leaves blocking gaps with actual Testule exit 5.

## Reviewed Go observation map

`make testule-observation-map` executes only the six reviewed Go-observable candidates above with `go test -race -count=1 -json`, without `-short`. One bounded stream is verified target-by-target with the same exact run/pass rules as the representative source, then retained with the exact checkout revision. This is still native source material only; no Testule fingerprint, normalized Evidence record, or gap state is assigned by this job.

The required `level.endToEnd` row is intentionally excluded. Its authoritative candidate is the built-process `make e2e` smoke, which is not Go JSON. That generic external-process ingestion need is recorded on Testule #16 rather than being hidden by relabeling an in-process Go test.

## Generation and deployment disposition

Example-based validation is required. Generated, property-based, **fuzz**, and model-based generation are optional in this first plan. `docs/ci.md` currently records **no Go fuzz target**: optional fuzz is an honest visible gap, not an inapplicability decision or completed evidence. Revisit its priority with an actual stable corpus and native bounded fuzz campaign. AI-assisted test generation is not an authoritative correctness requirement; independently verified AI-generated fixtures may still be useful. There is no currently deployed service/system boundary distinct from Repora's local integration and built-CLI E2E boundaries.

## Artifact and release evidence are separate

PR Linux verification binaries execute the CLI smoke; Windows/macOS binaries are cross-compiled but not run on those platforms. Nix package smoke checks its executable and installed support files. These observations **do not** verify a published release asset. Release candidate and post-publication evidence require exact commit/tag/asset identity, digest, archive contents, checksum verification, packaged-binary execution, and supported installation path as specified by `docs/release-checklist.md`. Do not satisfy a TestPlan row by relabeling an unrelated source-tree or cross-compilation check.

## Acceptance before enabling a Testule gate

1. The checked-in plan has already passed an actual pinned Testule `v1alpha1` validation; invalidate that proof if the plan bytes change and revalidate before claiming conformance.
2. Review the exact assertions and native events for each candidate. Record missing dimensions rather than attaching invented annotations.
3. Resolve the Go 1.25.13 / Testule Go 1.26 toolchain and distribution decision without silently raising Repora's supported Go version or adding an untrusted artifact source.
4. Preserve native correctness checks and run a representative intentionally incomplete Evidence smoke before a complete, exact-revision gap gate.
5. Qualify Nix/build outputs and released assets separately; update #178 and organization rollout #36 only with actual validation and merge evidence.
