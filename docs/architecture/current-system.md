# Current system architecture

Status: Current

This document describes merged Repora behavior. Future architecture belongs in proposals or ADRs until implemented.

## Product boundary

Repora is a local-first repository controller exposed primarily through the `repoctl` Go CLI. Implemented capabilities are separated into bounded domains rather than one generic mutation engine:

1. **Git-ref reconciliation** — observe, plan, dry-run, and reconcile canonical/default-branch state to configured mirrors.
2. **Managed README artifacts** — deterministically render, review, preflight, and apply exactly root `README.md` on the canonical default branch.
3. **Repository assessments** — create, validate, and project local evidence-backed assessment artifacts without Git/provider mutation.
4. **Document routing** — deterministic repository-owned routing/trust/context contracts used to select evidence without executing source or granting mutation authority.
5. **Repository/CI posture inventory** — GET-only GitHub observation that normalizes repository, protection, file, and Actions evidence without evaluating policy or mutating provider state.
6. **Documentation posture** — GET-only profile-driven document/README hygiene observation that preserves routing trust metadata without prose scoring, policy evaluation, or remediation authority.
7. **Hooks/local-workflow posture** — GET-only bounded hook/config/workflow observation that preserves CI authority and never installs or executes target-repository hook code.
8. **Mirror posture** — topology-driven canonical/mirror identity and drift observation that reuses existing reconciliation semantics, may refresh Repora's local cache, and never pushes or mutates provider settings.
9. **Packaging and assurance** — release archives plus a standalone Nix package/check/development surface that reuses canonical validation boundaries.

Repora does not provide arbitrary repository-file mutation, provider provisioning, hosted orchestration, automatic posture remediation, or a general policy engine.

## Git-ref reconciliation domain

For each repository entry Repora supports one GitLab canonical and one or more GitHub/GitLab mirrors, provider/path identity, runtime HTTPS resolution, default-branch-only policy, independent mirror status, exact reconciliation artifact v2, complete stale/OID preflight, sequential independent mirror mutation, reviewed force-with-lease overwrites, per-target apply v3 outcomes, immutable execution evidence, and bounded repository concurrency.

It does not provide tags, non-default branches, deleted-ref reconciliation, provider provisioning, rollback, or cross-remote transactions.

```text
configuration
  -> observe canonical and mirrors
  -> exact reconciliation artifact
  -> prepare all selected repositories
  -> require force authorization when needed
  -> persist INTENT
  -> validate every expected OID
  -> execute sequential independent mirror actions
  -> persist RESULT
```

Preparation failure prevents selected mutation. After a repository has passed complete preflight, independent mirror failures are recorded without rolling back earlier success or suppressing later independent attempts.

## Managed README domain

README management is an explicit per-repository capability under `artifacts.readme`. Its authority is fixed to repository-root `README.md` on the configured canonical default branch.

```text
contained local template + values
  -> deterministic render
  -> exact canonical observation
  -> managed-artifact plan v1 + review diff
  -> exact stale preflight / dry-run
  -> managed-artifact INTENT
  -> isolated verified candidate commit
  -> fresh preflight
  -> exact reviewed-base leased canonical push
  -> managed-artifact RESULT
  -> separate fresh mirror status -> plan -> apply when desired
```

Templates cannot execute code or fetch remote content. Existing regular README mode is preserved, unsafe/non-regular state fails closed, real apply does not edit a user checkout, and there is no force override. Multi-repository mutation is explicitly non-transactional and evidenced per repository.

## Assessment domain

Current local assessment commands validate `repora.repository-assessment` v1, project findings/scorecards, and exclusively create the canonical skeleton. They do not load repository topology, call Git/provider APIs, infer findings, or mutate repositories. Assessment reports remain point-in-time evidence rather than live work-state authority.

## Document-routing domain

Repository-owned `.repora/` routing contracts support deterministic manifests, trust tiers, context receipts, hierarchical summaries, and bounded Go AST selectors. Routing does not execute selected source, perform semantic/vector retrieval, infer ownership, or grant mutation authority.

## Shared posture fact model

Posture fact contracts distinguish:

- `observed` — source evidence was available and the value was actually seen, including observed boolean `false`;
- `unknown` — the fact is representable but current evidence/scope cannot establish a value;
- `unavailable` — relevant evidence could not be read or safely established under the current observation boundary.

Fact collection remains separate from policy evaluation. Current collectors do not assign severity, create findings, execute scanners, authorize remediation, or mutate provider settings.

## Repository/CI posture inventory domain

`repoctl posture inventory OWNER/REPO` emits `repora.posture-inventory` v1 JSON for a GitHub repository.

Current facts include default branch, branch-protection summary/detail where available, required status checks/reviews, force-push/deletion protection, supported CODEOWNERS/SECURITY/license/template/dependency-automation paths, workflow paths, workflow/job permissions, `pull_request_target`, runner labels/literal `self-hosted` evidence, and action/reusable-workflow references with pinning classification.

Important boundaries:

- the collector depends on a `GitHubReader` exposing only repository, branch, branch-protection, tree, and blob reads;
- the production HTTP adapter issues only GET requests;
- optional `GITHUB_TOKEN`/`GH_TOKEN` values are environment-only runtime inputs and are not persisted in evidence;
- provider-hidden 401/403/404 data becomes unavailable rather than a false negative;
- Git tree truncation preserves known-present facts but makes unsupported negative claims unknown;
- workflow YAML is treated as bounded untrusted data;
- no policy, finding, scanner, issue/PR generation, branch-protection update, or provider mutation is reachable from inventory.

## Documentation posture domain

`repoctl posture docs OWNER/REPO` emits `repora.posture-documentation` v1 JSON while preserving the shared fact states.

A repository may declare deterministic observation targets in `.repora/posture-documentation.yaml` using `repora.posture-documentation-profile` v1. The profile selects facts to observe; it does not assign severity, suppress external policy, create findings, or grant remediation authority.

Current documentation facts include configured document presence, README ATX-heading/link presence, exact configured content markers, and document-router trust-tier evidence. The collector preserves generated/archived/canonical authority rather than treating every document as equivalent.

Important boundaries:

- it reuses the GET-only GitHub reader and default-branch tree/blob evidence;
- a built-in baseline observes only root `README.md` when a profile is known absent from a complete tree;
- truncated/inaccessible/malformed evidence becomes unknown or unavailable rather than a passing fact;
- repository-controlled profile/router/Markdown input is bounded data and is never executed;
- full prose linting, semantic/LLM judgment, policy evaluation, reports, rewrites, and provider mutation remain outside the domain.

## Hooks/local-workflow posture domain

`repoctl posture hooks OWNER/REPO` emits `repora.posture-hooks` v1 JSON while preserving the shared fact states.

A repository may declare bounded observation expectations in `.repora/posture-hooks.yaml` using `repora.posture-hooks-profile` v1. The profile may name a manager, additional hook paths, required local checks, bootstrap documentation, and bypass/escape-hatch documentation. It is observation configuration only and cannot assign severity, suppress policy, create findings, or grant remediation authority.

Current hooks/local-workflow facts include:

- common manager/config signals for pre-commit, Lefthook, Husky, and generic `.githooks` layouts;
- configured hook-entrypoint presence;
- required local-check declarations and whether each check name is observable in GitHub Actions workflow text;
- bootstrap and bypass-document presence where explicitly declared;
- bounded static network-load signals in hook/config blobs;
- explicit executable state, currently `unknown` because the shared GitHub tree model does not normalize file mode.

Important boundaries:

- it reuses the GET-only GitHub reader and immutable default-branch tree/blob evidence;
- hook, profile, and workflow inputs are treated as bounded untrusted data;
- target-repository hooks, generated scripts, package managers, and hook-manager binaries are never installed, sourced, executed, or bootstrapped;
- network references are only detected as static signals and are never followed;
- apparent CI coverage is a deterministic text observation, not proof of semantic equivalence;
- CI remains the enforcement authority unless a future explicit policy states otherwise;
- truncated/inaccessible/malformed evidence becomes unknown or unavailable rather than a passing fact;
- policy evaluation, findings, remediation, and provider mutation remain outside the domain.

## Mirror posture domain

`repoctl posture mirrors -f repora.yaml` emits `repora.posture-mirrors` v1 JSON for repositories already declared in Repora topology.

Mirror posture records:

- repository ID and durable UID;
- configured `mirror` mode and `canonical_to_mirror` direction;
- canonical and mirror `provider:path` identities;
- Repora cache remote names;
- canonical and mirror default-branch names where observable;
- default-branch commit evidence;
- default-branch-name drift;
- existing `EQUAL`, `BEHIND`, `AHEAD`, and `DIVERGED` reconciliation state plus ahead/behind counts;
- GitHub visibility and authenticated/current-actor push permission when returned by the GET-only repository endpoint;
- tag and release drift as explicit `unknown` facts under the current default-branch-only scope.

Mirror drift is not recalculated independently. The collector reuses `status.CheckAll`, preserving the existing reconciliation algorithm and per-mirror partial-error semantics.

Observation can create or refresh Repora's local bare mirror cache, configure cache remotes, and fetch repository state. That local observation boundary is not side-effect-free, but it does not call push, synchronization, release publication, or provider-setting mutation operations.

A failed mirror does not cause cached remote HEAD data to be treated as current. Missing/unavailable branch evidence therefore cannot produce an observed healthy/default-branch-drift result by inference. Independent GitHub metadata may establish a branch name even when local branch evidence is unavailable, while commit/reconciliation evidence remains unavailable.

GitLab Git transport/reconciliation evidence is supported; GitLab provider-administration metadata remains unavailable until a posture adapter exists. GitHub actor permissions are identity-specific and are therefore exposed as `current_actor_push_permission`, not a universal writeability claim.

## Package ownership

| Package/surface | Owns | Must not own |
| --- | --- | --- |
| `internal/config` | strict YAML, durable identity, topology/ref-policy normalization, managed README opt-in | Runtime Git/provider mechanics |
| `internal/refpolicy` | closed versioned ref scope and intent decisions | Git operations or command authorization |
| `internal/transport` | runtime provider/path URL resolution | Durable identity or policy |
| `internal/status` | canonical/mirror observation and divergence evidence | Mutation decisions or pushes |
| `internal/plan` | deterministic reconciliation actions | Git reads/writes or durable serialization |
| `internal/planartifact` | exact reconciliation artifact parsing/validation/compatibility | Observation or execution policy |
| `internal/executor` | OID preflight, bindings, pushes, leases, action outcomes | Recomputing status or policy |
| `internal/apply` | reconciliation binding, force authorization, audit orchestration, apply results | Implicit target selection or rollback |
| `internal/managedartifact` | README template/render/observation/plan/preflight/candidate verification | Generic file authority or mirror reconciliation |
| `internal/managedartifactapply` | journaled managed README execution/result correlation | Replanning reviewed content |
| `internal/assessment` | strict assessment contracts, validation, skeleton, projections | Live discovery, automated scoring, mutation |
| `internal/posture` | versioned posture facts and bounded repository/CI, documentation, hooks/local-workflow, and mirror observation | Policy evaluation, findings, scanners, remediation, provider mutation |
| `internal/journal` | immutable intent/result evidence and protected persistence | Mutation or replay authority |
| `internal/git` | bounded Git subprocess/cache/object/ref/push mechanics | Product policy or durable identity |
| `.repora/` + routing/posture profile validators | deterministic route/trust/context and documentation/hooks observation contracts | Source execution, severity policy, or mutation authority |
| `cmd/repoctl` | command routing, output contracts, bounded orchestration | Duplicated domain mechanics |

## Identity, authorization, and evidence

Repora distinguishes human `id`, durable logical `uid`, durable `(provider,path)` target identity, deterministic configuration order, and ephemeral resolved URLs/remote aliases.

Mirror destructive intent requires existing local policy plus explicit `--force`. Managed README apply is a separate authority domain and accepts no force override. Repository/CI, documentation, and hooks posture expose no mutation authority; mirror posture can refresh only Repora's observation cache and exposes no push/synchronization/provider-mutation capability.

ADR-0018 defines a future optional Anthesis `pre_apply` authorization seam after Repora local preparation but before Git execution INTENT/mutation. No runtime evaluator, transport, CLI/config integration, or approval workflow is currently implemented.

Journals and assessment/posture artifacts are evidence, never replay authority. Retry after mutation/stale state requires fresh observation and a new exact plan.

## Public contracts

Current serialized contracts include:

- `repora.status` v2;
- exact reconciliation artifact v2 with historical v1 support;
- `repora.apply` v2/v3 compatibility surfaces;
- execution-record v3 with historical parsers;
- `repora.io/managed-artifact-plan` v1;
- `repora.io/managed-artifact-apply-result` v1;
- `repora.io/managed-artifact-execution-record` v1;
- `repora.repository-assessment` v1 and linked evidence/scorecard schemas;
- `repora.posture-inventory` v1;
- `repora.posture-documentation` v1;
- `repora.posture-documentation-profile` v1;
- `repora.posture-hooks` v1;
- `repora.posture-hooks-profile` v1;
- `repora.posture-mirrors` v1.

Versioned files under `schemas/` are authoritative for serialized shapes.

## Packaging and assurance boundary

`v0.1.0` established the published mirror-controller baseline through GitHub Release archives plus SHA-256 checksums. The repository also owns a standalone Nix flake for supported Linux/macOS systems; its checks reuse canonical formatting, unit, integration, contract, E2E, and static-analysis boundaries.

Assurance includes race-enabled tests, contract/CLI E2E tests, Go vet and Staticcheck, immutable workflow action pins, least-privilege permissions, govulncheck, CodeQL, full-history secret detection, dependency-license validation, release-package reproduction, and scheduled deep validation.

A repository-wide benchmark gate, release signing, and full provenance attestation remain deferred.
