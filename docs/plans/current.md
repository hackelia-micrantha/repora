# Current implementation plan

Status: Active
Reviewed: 2026-10-30
Authority: detailed acceptance and evidence live in GitHub issues; this page is a current sequence, not a substitute for exact-head verification.

## Released and implemented boundary

- **Published:** `v0.2.0` (2026-09-02) remains the earlier standalone acceptance baseline; `v0.3.0` was published on **2026-10-24** from exact commit `2213693597e5fe3915f685840fb75f8103a5191f`. Independent published-asset checks (all four archives/checksums), packaged Linux CLI version/commit smoke, and real public GitLab→GitHub status/plan/dry-run with no provider write are recorded in completed #218.
- **Never published:** the historical `v0.2.1` candidate at `6fe22ee2c7accb36e60d2df78fbe3ac59ba48616`. #183 was superseded rather than retargeting or publishing this immutable candidate.
- **Published v0.3.0 feature surface:** complete Nix/Unix support files; assumption-free local bootstrap lifecycle and read-only discovery; local-only Git storage posture; CI-environment posture v1/v2; flake-first evidence; offline posture-policy conditional applicability v2; and the bounded six-target native Testule observation-map source merged by #196. That source is not normalized Testule Evidence, does not satisfy built-process E2E ingestion, and is not complete Testule qualification. Publication is not evidence that the future control plane is deployed.
- **Security chain:** #216 / PR #217 first patched the compiler to Go 1.26.9; the new GO-2026-6809 advisory required Go 1.26.10, delivered by #223 / merged PR #224 at `f0f092226d4344cf1f9a0eeb27ff770409cda551`. Receiving-main CI, security, shared Nix and Mise checks passed. Candidate-specific release packaging, patched binary provenance and installed acceptance remain release gates. Previously published archives are not retroactively patched.
- **End-to-end acceptance:** #211 Gate A is complete from #138 published-binary acceptance plus preserved regression evidence. #221 remains a separate P1 private/private credentialed pilot, not a v0.3 blocker. Gate B governed effects and Gate C broader estate support remain independent future milestones; ADR-0019 is proposed, not runtime authority.

The supported mirror contract remains a **GitLab canonical**, one or more **GitHub, GitLab, or Bitbucket Cloud mirrors**, **default branch only**, exact-plan/OID preflight and honest partial results. Managed README mutation and read-only posture/reporting are separate bounded domains.


## Repository bootstrap and template boundary

Repository bootstrap is implemented, but policy, transport, and execution authority remain deliberately separated:

1. **`hackelia-micrantha/.github` owns organization policy and the bootstrap schema.** Resolution semantics, decision provenance, default-branch policy, CI/security/release standards, and other organization requirements originate there.
2. **`hackelia-micrantha/repository-template` is a thin optional creation transport.** Its generic materialized state is limited to bootstrap instructions plus `.repora/bootstrap.proposed.json`; only the organization-authorized `main` default branch is pre-resolved. It must not add a license, language/runtime, package manager, placeholder flake, CI/release workflow, product scaffold, project CODEOWNERS, or copied organization-inherited community files merely because a repository is being created.
3. **Repora owns the bootstrap lifecycle and later reconciliation.** `repoctl bootstrap init → inspect → plan → apply` consumes explicit decisions, binds exact reviewed artifacts, revalidates stale state, and applies only bounded supported local effects. `bootstrap discover` observes existing/new repositories without turning evidence into decisions or mutation authority.
4. **Template provenance is evidence, not policy.** A template-derived repository must still pass normal Repora inspection/convergence. Template updates do not automatically propagate, and template freshness is not itself a compliance requirement unless an applicable policy explicitly requires convergence.

Provider-side repository creation remains a separate future capability. If implemented, it must consume resolved bootstrap decisions, bind the exact template/revision when used, pass visibility and branch semantics explicitly, and continue through normal Repora verification after creation.

## Completed release — #218, v0.3.0

- `v0.3.0` was tagged and [published](https://github.com/hackelia-micrantha/repora/releases/tag/v0.3.0) from the exact reviewed receiving-main commit `2213693597e5fe3915f685840fb75f8103a5191f`.
- Required exact-commit CI/security/Nix/Mise, deep validation, and pre-tag repeat packaging were qualified before publication. The published four-platform archives independently matched their checksum manifest; each contains the supported distribution files and patched Go 1.26.10 binary version marker. Linux amd64 was actually executed; macOS and Windows were cross-built/inspected, not natively executed.
- The published Linux `repoctl` reported the exact tag/source commit and completed live public GitLab→GitHub status/plan/exact-plan dry-run without remote mutation. A public-provider smoke is **not** the private/private pilot.
- #218 is **closed as shipped and verified**; P0 release priorities were reconciled before publication. The archive checksum manifest is an integrity check, not signed publisher provenance.

## Immediate next operator gate — #221, private/private real mirroring

1. Reuse the previously selected verified private/private pair, not an inferred same-name match. Provider API `main` equality is a point-in-time observation, not native Git transport acceptance.
2. In an approved isolated operator/service context, provision narrow noninteractive read-only GitLab/GitHub transport principals, with secrets kept outside agent/Desktop Commander, `repora.yaml`, Nix store and public logs.
3. Run independently authenticated `git ls-remote` against both fixed endpoints, then the installed `v0.3.0` `repoctl status → exact plan → apply --dry-run`. Retain binary/config hashes, OIDs, exit codes and sanitized evidence; independently re-observe target heads.
4. Close #221 only when the native private/private pilot evidence and negative-path coverage are qualified. Any real provider push, canonical-direction decision, or native mirror-setting change is separately authorized; no automatic whole-ref sync is implied.

## Next milestones (not v0.3.0 blockers)

1. **Governed default-branch reconcile:** #211 Gate B, #212 design, #213 runtime. Gateway actor identity/freshness, Anthesis authorization, sealed effector credentials and proof of no direct bypass must be resolved before any real governed write; proposed ADR-0019 is not runtime authority.
2. **Provider-neutral CI/CD control plane:** #160 and #184–190, starting with topology-aware attachments and read-only inventory; provider workflow engines remain execution authority.
3. **Repository topology and trust domains:** #149–156, including Forgejo, projections, promotions, proposals, and archive/restore. Each relationship has its own trust boundary; do not infer reverse sync or private history publication.
4. **Additional Git scope and local workflows:** #215 multi-ref policy design; #147 workspace bootstrap; #166 local storage hygiene. No unreviewed destructive ref or history rewrite.
5. **Evidence improvements:** #178/#180 Testule consumer/E2E qualification after completed native-map slice #195, #182 release posture, and provenance/signing #157. Never treat staged native output as complete normalized Testule evidence.

P0/P1 labels within a long-range product workstream must not implicitly make that work a `v0.3.0` release blocker.

## Execution and simplicity constraints

- Preserve durable UID and provider/path mirror target identity, exact stale-safe plans, bounded policy and immutable INTENT/RESULT evidence.
- Keep the standalone trusted CLI distinct from a future Gateway-governed deployment; packaging or a policy report does not grant mutation authority.
- Preserve observed, unknown and unavailable facts. Repository-owned observation profiles do not assign policy severity or authorize changes.
- No arbitrary shell/proxy Gateway, runtime credential leakage, implicit force push, cross-remote atomicity or automatic rollback claim.
- Reuse repo-owned flake/toolchain quality gates and existing collectors rather than parallel policy/testing engines.
- A completed slice requires independent review, current-head checks, negative tests where applicable, updated contracts/docs and operator evidence suitable to the actual authority boundary.
