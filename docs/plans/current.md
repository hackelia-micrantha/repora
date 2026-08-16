# Current implementation plan

Status: Active

## Current objective

Repora's released mirror controller and first post-v0.1 managed-artifact, routing, assessment, policy-design, standalone packaging, read-only GitHub posture-inventory, and deterministic documentation-posture foundations are complete. The current objective is to extend the normalized posture fact model across mirror, local-workflow, and commit/process domains before adding policy evaluation or reporting.

## Completed foundation

| Area | State | Notes |
| --- | --- | --- |
| Durable `uid` identity | Complete | Cache and durable artifacts use logical identity. |
| Provider/path topology | Complete | Multiple mirrors use stable provider/path identity. |
| Ref policy v1 | Complete | Default branch only; destructive mirror actions require explicit authorization. |
| Multi-mirror status/planning | Complete | Exact artifact v2 is provider/path-bound and deterministic. |
| Stale-safe mirror execution | Complete | Complete OID preflight, exact leases, partial-result evidence, and no implicit rollback. |
| Execution evidence | Complete | Immutable Git-ref INTENT/RESULT journals preserve reviewed intent and outcomes. |
| v0.1 release | Complete | `v0.1.0` tag and published assets were independently verified. |
| Managed README lifecycle | Complete | Deterministic config/template/render/plan/dry-run/apply with exact stale preflight, leased canonical push, and durable evidence. Mirror propagation is a separate fresh reconciliation cycle. |
| Managed README example | Complete | Repository-contained configuration/template example is regression-tested for deterministic rendering. |
| Document routing foundation | Complete | Deterministic routes, trust tiers, manifests, context receipts, hierarchical summaries, and bounded Go AST routing. |
| Repository assessment | Complete | Evidence-backed assessment schemas, validation, skeleton creation, finding projection, and scorecard projection are implemented. |
| Managed-artifact Go contract consolidation | Complete | `internal/managedartifact` is the single authoritative plan-v1 implementation. |
| Optional Anthesis integration design | Complete | ADR-0018 selects additive `pre_apply`; runtime evaluator/transport remains deferred. |
| Standalone Nix packaging | Complete | Repository-owned package/app/check/dev-shell/formatter outputs reuse canonical validation and require no Dubnium access. |
| Test pyramid/static analysis | Complete | Fast/unit, integration, contract, CLI E2E, race/deep validation, Go vet, and Staticcheck are explicit repository/CI boundaries. |
| GitHub posture inventory v1 | Complete | GET-only normalized repository/CI facts preserve observed, unknown, and unavailable evidence; no mutation-capable provider boundary is exposed. |
| Documentation posture v1 | Complete | Profile-driven document/README/link/content-marker facts reuse the posture evidence model and preserve routing trust tiers without prose scoring or remediation authority. |

## Active sequence

### 1. Add mirror-management drift facts (#120)

Reuse existing provider/path topology and mirror semantics rather than creating a separate mirror scanner.

Exit condition:

- canonical/mirror identities and default branches are normalized into posture evidence;
- drift, visibility/writeability, and release/tag evidence preserve unavailable provider state explicitly;
- no synchronization or provider mutation occurs;
- tests cover in-sync, drift, missing, and unavailable evidence.

### 2. Add local workflow/hook and commit/process fact domains (#123, #122)

Add independent bounded sources after the shared model is proven:

- hooks/local workflow facts (#123) without installing or executing hook code;
- bounded commit/process-risk facts (#122) without productivity scoring, identity profiling, or intent inference.

These slices may proceed independently where their dependencies permit and must not bypass the shared evidence representation.

### 3. Converge facts into explainable policy/reporting (#121)

Only after the source fact contracts are proven should Repora evaluate them into deterministic posture findings and Markdown reports.

Exit condition:

- profiles express expected vs observed state with explicit severity;
- unknown/unavailable evidence remains visible;
- exceptions require reason, owner, and expiry;
- reports are deterministic and evidence-backed;
- policy/reporting consumes normalized facts and does not rescan providers itself;
- no provider mutation is introduced.

### 4. Reassess runtime policy integration only on concrete demand

ADR-0018 defines the optional Anthesis `pre_apply` seam, but design completion does not imply runtime implementation priority.

Resume only when there is a concrete evaluator contract/deployment path and operator need. If resumed, the first implementation slice remains transport-free and mutation-neutral: versioned policy facts/decision schemas, deterministic facts construction, evaluator interface/fakes, and local policy evidence before inserting a live gate.

## Explicit deferrals

- tags, non-default branches, wildcard refspecs, and deleted-ref reconciliation;
- concurrent mirror mutation;
- automatic rollback or cross-repository/cross-remote transactions;
- Anthesis runtime transport/authentication and approval workflows;
- provider provisioning or hosted control-plane behavior;
- automatic provider-setting remediation from posture findings;
- automatic managed-artifact mirror propagation;
- arbitrary managed file generation;
- package-manager publication beyond the repository-owned Nix flake unless separately prioritized;
- containers, release signing, and full provenance attestation;
- repository-wide performance gates without a stable workload and useful threshold.

Deferred tracks must reuse current identity, plan, policy, execution, result, evidence, test, and static-analysis substrates rather than create parallel paths.

## Simplicity constraints

- Preserve Repora as a standalone local-first controller.
- Prefer vertical capabilities over disconnected internal models.
- Keep Git-ref reconciliation, managed artifacts, routing, assessments, posture, and optional external policy as separate bounded domain contracts.
- Reuse durable `uid`, `provider:path`, and shared posture/assessment evidence concepts across posture work.
- External policy may add authorization but may never rewrite or weaken reviewed Repora intent.
- Keep ref-policy v1 closed and mirrors sequential until a separate decision expands them.
- Do not imply cross-remote atomicity, rollback, provider remediation, or approval semantics that are not implemented.
- Packaging may expose capability; it must not silently grant mutation authority.
- CI/Nix/posture must inspect or reuse canonical validation rather than redefine it independently.

## Definition of done

A slice is complete only when observable behavior, appropriate failure/security tests, versioned contracts where applicable, documentation, issue acceptance criteria, independent review, and exact-head CI/security validation are complete.

For design-only slices, source behavior must remain unchanged and the implementation boundary/deferred work must be explicit.

## Maintenance

Update this plan when a capability completes, implementation order changes, or deferred work is explicitly reprioritized. GitHub issues remain authoritative for detailed acceptance criteria and live work state.
