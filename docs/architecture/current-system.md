# Current system architecture

Status: Current

This document describes merged Repora behavior. Future architecture belongs in proposals or ADRs until implemented.

## Product boundary

Repora is a local-first repository controller exposed primarily through the `repoctl` Go CLI. Its implemented capabilities are intentionally separated into bounded domains rather than one generic mutation engine:

1. **Git-ref reconciliation** — observe, plan, dry-run, and reconcile canonical/default-branch state to configured mirrors.
2. **Managed README artifacts** — deterministically render, review, preflight, and apply exactly root `README.md` on the canonical default branch.
3. **Repository assessments** — create, validate, and project local evidence-backed assessment artifacts without Git/provider mutation.
4. **Document routing** — deterministic repository-owned routing/trust/context contracts used to select evidence and context without executing source or granting mutation authority.
5. **Packaging and assurance** — release archives plus a standalone Nix package/check/development surface that reuses canonical validation boundaries.

Repora does not provide arbitrary repository-file mutation, provider provisioning, hosted orchestration, or a general policy engine.

## Git-ref reconciliation domain

For each repository entry Repora supports:

- one GitLab canonical repository;
- one or more GitHub/GitLab mirrors for status, exact planning, dry-run, and real mutation;
- provider-relative `provider + path` topology with bounded single-mirror legacy URL compatibility;
- stable target identity as `provider:path`;
- runtime HTTPS transport resolution;
- default-branch-only closed ref policy;
- independent `EQUAL`, `BEHIND`, `AHEAD`, `DIVERGED`, or `ERROR` status per mirror;
- provider/path-bound reconciliation artifact v2 across all required actions;
- historical single-mirror artifact v1 compatibility through the legacy execution path;
- complete topology, policy, branch, and OID preflight before action zero;
- sequential independent mirror mutation with continuation after runtime failure;
- normal pushes and explicitly authorized force-with-lease overwrites;
- apply v3 per-target outcomes for mixed or multi-mirror selections;
- fail-closed execution-record v3 intent/result evidence;
- bounded repository-level concurrency.

It does not provide tags, non-default branches, deleted-ref reconciliation, provider provisioning, rollback, or cross-remote transactions.

### Mirror runtime flow

```text
configuration
  -> observe canonical and all mirrors
  -> match mirrors by provider:path
  -> build or import exact artifact v2
  -> prepare every selected repository
  -> require command-level force authorization when needed
  -> for each repository:
       validate topology, policy, state/action intent, and default branches
       persist execution-record v3 INTENT
       validate every expected source and target OID
       if dry-run: persist VALIDATED/STALE/SKIPPED RESULT
       if real:
         execute actions sequentially in artifact order
         continue after independent runtime failures
         persist APPLIED/FAILED outcomes in RESULT
  -> render apply v3 results and deterministic exit status
```

Preparation failure in any selected repository prevents all selected mutation. Once repository execution begins, repositories are independent and may run concurrently. Mirrors inside one repository execute sequentially.

## Managed README domain

README management is an explicit per-repository capability under `artifacts.readme`. It is separate from Git-ref reconciliation and cannot target arbitrary paths.

The supported managed path is exactly repository-root `README.md` on the configured canonical default branch. The lifecycle is:

```text
local contained template + configured values
  -> deterministic render
  -> fresh canonical README observation
  -> exact managed-artifact plan v1 + review diff
  -> operator review
  -> exact stale preflight / optional dry-run
  -> durable managed-artifact INTENT
  -> isolated verified candidate commit
  -> fresh all-target preflight
  -> exact reviewed-base leased canonical push
  -> managed-artifact RESULT
  -> separate fresh mirror status -> plan -> apply when propagation is desired
```

Important boundaries:

- templates are configuration-root-relative bounded regular files;
- rendering is a small single-pass token substitution language with no executable functions, includes, plugins, environment interpolation, network access, or recursive expansion;
- existing regular README mode is preserved; a missing README is created as `100644`;
- symlink, submodule, tree, oversized, unsafe-text, or otherwise non-regular README state fails closed;
- exact desired README bytes and target/base state are reviewed before mutation;
- real apply creates candidate Git objects in Repora's cache rather than editing a user checkout;
- no `--force` override exists for managed README apply;
- every push uses the exact reviewed base as its lease;
- multi-repository mutation can partially succeed and records that outcome explicitly;
- canonical README mutation invalidates earlier mirror observations/plans; mirror propagation is deliberately a separate fresh review cycle.

## Assessment domain

Repository assessment artifacts are point-in-time evidence, not a replacement source of truth for GitHub state.

Current CLI support includes:

- `validate-report FILE` — strict local validation of `repora.repository-assessment` v1;
- `list-findings FILE` — deterministic projection of validated findings;
- `generate-scorecard FILE` — deterministic projection of recorded scorecard dimensions without recalculation;
- `assess FILE` — exclusive creation of the canonical assessment skeleton without overwrite or parent-directory creation.

These commands do not load repository topology, call Git, call provider APIs, infer findings, or mutate repositories. Assessment evidence may reference repository/GitHub sources, but live work state remains authoritative in those systems.

## Document-routing domain

Repora contains deterministic routing contracts under `.repora/` and supporting validation tooling for evidence/context selection. Current foundations include:

- route manifests and deterministic matching fixtures;
- trust tiers and explicit lower-trust inclusion;
- context receipts;
- summary-first hierarchical routing;
- bounded Go AST source selectors that refine an already eligible route candidate set.

Routing does not execute selected source, perform semantic/vector search, infer ownership, or grant mutation authority. Summary and generated artifacts remain derived evidence unless an explicit route includes their trust class.

## Package ownership

| Package/surface | Owns | Must not own |
| --- | --- | --- |
| `internal/config` | strict YAML, durable identity, safe endpoint paths, topology/ref-policy normalization, managed README opt-in, duplicate target rejection | Runtime URL derivation or Git operations |
| `internal/refpolicy` | closed versioned ref scope and relationship-to-intent decisions | Git operations or command authorization |
| `internal/transport` | runtime provider/path URL resolution | Durable identity or policy |
| `internal/status` | canonical/mirror observation, target identity, divergence, and commit evidence | Mutation decisions or pushes |
| `internal/plan` | deterministic reconciliation actions and compatibility projection | Git reads/writes or durable serialization |
| `internal/planartifact` | versioned exact reconciliation artifact parsing, provider-path validation, historical compatibility, and plan conversion | Observation or execution policy |
| `internal/executor` | complete OID preflight, runtime bindings, sequential independent pushes, leases, and action outcomes | Recomputing status or policy |
| `internal/apply` | reconciliation artifact construction, topology/status/policy binding, force authorization, audit orchestration, and apply v2/v3 results | Implicit target selection or rollback policy |
| `internal/managedartifact` | README template loading/rendering, exact observation/planning/preflight contract, candidate verification, and managed-plan v1 parsing/validation | Generic file authority or mirror reconciliation |
| `internal/managedartifactapply` | journaled managed README execution ordering and per-repository result correlation | Replanning reviewed content or generic Git-ref apply |
| `internal/assessment` | strict assessment parser/validator, skeleton, finding/scorecard data contracts | Live repository discovery, automated scoring, or mutation |
| `internal/journal` | immutable path-bound intent/result evidence, digest correlation, redaction, and local persistence | Mutation or replay authority |
| `internal/git` | bounded Git subprocesses, cache safety, refs, object plumbing, pushes, leases, timeouts, and redaction | Product policy or durable identity |
| `.repora/` + routing validation scripts | deterministic route/trust/context contracts and fixtures | Source execution, semantic retrieval, or mutation authority |
| `cmd/repoctl` | command routing, preparation aggregation, concurrency, output versions, artifact/report I/O, and exit semantics | Git mechanics or duplicated planning |

## Identity and runtime binding

Repora distinguishes:

- `id`: human-facing repository alias;
- `uid`: durable logical repository identity;
- `(provider, path)`: durable repository or mirror selector;
- configuration index: deterministic order only;
- resolved URL and Git remote alias: ephemeral runtime state.

Status v2, reconciliation artifact v2, apply v3, and execution-record v3 use provider/path identity. Managed-artifact plan v1 also binds canonical provider/path/branch and exact base OID. Runtime aliases and local template/cache paths are never durable execution identity.

## Reference policy and authorization

Ref-policy v1 supports exactly:

- `scope: default-branch-only`;
- `destructive: require-force`.

Mirror planning records forced intent for ahead or diverged mirrors. A real mirror command containing any forced action requires `--force` before journal creation or mutation. The flag authorizes only actions already marked forced. Dry-run validates forced actions without authorization.

Managed README apply is a different authority domain: it accepts only its exact reviewed plan and never accepts `--force`.

ADR-0018 defines a future optional Anthesis `pre_apply` authorization seam after local Repora preparation but before Git execution INTENT/mutation. No runtime evaluator, policy transport, CLI/config integration, or approval workflow is currently implemented. If later enabled, external policy may add authorization but cannot weaken local Repora policy, stale checks, force requirements, or leases.

## Preparation, preflight, and stale state

### Mirrors

Before any selected mirror repository mutates, Repora completes observation and exact-artifact preparation for every selected repository. Artifact version, repository cardinality, topology, policy, current status, default branches, and action intent are bound before execution. After fail-closed INTENT persistence, executor preflight validates every expected source and target OID before action zero.

Any stale mirror action prevents every push for that repository, marks the offending action `STALE`, leaves unattempted actions `SKIPPED`, and still attempts result persistence.

### Managed README

Managed README preflight binds every planned UID to current configuration, requires README authority to remain enabled, requires canonical target/branch/base/content/mode state to match the reviewed plan, and recomputes the review diff. A mismatch is stale and fails before remote mutation.

Candidate creation has its own verification, and the pusher performs a fresh full preflight immediately before the sequential leased pushes. The exact lease closes the final remote race window.

## Independent mutation semantics

Mirror actions execute sequentially in exact artifact order after complete repository preflight. Runtime failure does not roll back earlier successful mirror actions and does not prevent later independent actions from being attempted. `APPLIED, FAILED, APPLIED` is a valid result.

Managed README multi-repository apply similarly does not claim atomicity. Earlier successful canonical pushes remain visible if a later repository fails. Retry in either domain requires fresh observation and a new exact plan; durable journals are evidence, not replay authority.

## Execution evidence

Git-ref artifact v2 execution writes execution-record v3 under `.repora/journal/`. Managed README apply writes its separate `repora.io/managed-artifact-execution-record` v1 INTENT/RESULT records through the same protected no-overwrite persistence substrate.

Intent persistence failure prevents the corresponding mutation path. Result persistence failure remains a command failure even if remote mutation already completed; output preserves the projected remote outcome so successful mutation is not hidden.

## Public contracts

Current public envelopes include:

- `repora.status` v2;
- compatibility `repora.plan` v1 for the legacy view;
- exact reconciliation artifact v2, with artifact v1 historical support;
- `repora.apply` v2 for single-mirror-only command selections;
- `repora.apply` v3 for mixed or multi-mirror selections;
- execution-record v3, with v1/v2 historical parsing support;
- `repora.io/managed-artifact-plan` v1;
- `repora.io/managed-artifact-apply-result` v1;
- `repora.io/managed-artifact-execution-record` v1;
- `repora.repository-assessment` v1 and its linked evidence/scorecard schemas.

Schemas under `schemas/` remain authoritative for serialized shapes.

## Concurrency and atomicity

Selected mirror repositories use bounded concurrency after global preparation and force authorization. Mirrors inside one repository are sequential.

Managed README repository pushes are also intentionally non-transactional and preserve per-repository outcomes. There is no cross-repository, cross-remote, or cross-domain transaction and no automatic rollback.

## Packaging and assurance boundary

`v0.1.0` established the published mirror-controller baseline through GitHub Release archives for Linux amd64, macOS amd64/arm64, and Windows amd64 plus SHA-256 checksums. The published assets were independently verified.

The repository now also owns a standalone Nix flake for `x86_64-linux`, `x86_64-darwin`, and `aarch64-darwin`. It exports package/app/check/dev-shell/formatter surfaces and reuses the repository's canonical formatting, unit, integration, contract, E2E, and static-analysis boundaries. Nix composition does not grant runtime Git or provider authority.

Security/quality assurance includes:

- race-enabled fast and integration tests;
- contract and CLI end-to-end tests;
- Go vet and Staticcheck;
- workflow syntax/policy validation with immutable action pins and least-privilege permissions;
- reachable-vulnerability scanning;
- CodeQL;
- full-history secret detection;
- dependency-license validation;
- reproducible release-package verification;
- scheduled deep validation and supported Go-version compatibility checks.

A repository-wide benchmark gate remains deferred until a stable workload and useful threshold exist. Release signing and full provenance attestation are also deferred.

## Current next boundary

The mirror controller, first managed README lifecycle, routing/assessment foundations, optional Anthesis integration design, and standalone Nix packaging are complete.

The next implementation track is the read-only repository/CI posture fact inventory under issue #118. It must reuse existing durable identity, topology, assessment/evidence, test, and static-analysis contracts rather than create a parallel control plane. Provider mutation and Anthesis runtime coupling remain deferred.
