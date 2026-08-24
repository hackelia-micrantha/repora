# Current implementation plan

Status: Active

## Current objective

Repora's v0.1 mirror-controller baseline is published, and current `main` contains a coherent post-v0.1 capability set: managed README mutation, routing and assessment foundations, standalone Nix packaging, repository posture collectors, offline posture convergence/policy/reporting, and Bitbucket Cloud mirror transport.

Project-truth reconciliation and representative operator acceptance are complete. The immediate milestone is to publish and independently verify the `v0.2.0` pre-alpha capability baseline before selecting additional product scope.

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
| Mirror posture v1 | Complete | Declared canonical/mirror identities, default-branch names, existing reconciliation drift, and bounded provider metadata facts reuse topology/status semantics without provider mutation or expanded ref scope. |
| Hooks/local-workflow posture v1 | Complete | Common/custom hook signals, declared local checks, CI-coverage evidence, bootstrap/bypass documentation, and static network-load signals are normalized without installing or executing repository hook code. |
| Bounded commit-history posture v1 | Complete | A capped default-branch history window exposes signature, merge-shape, size/file-scope, sensitive-path, and commit/PR-association evidence without identity analytics, productivity scoring, blame, or intent inference. |
| Posture convergence/policy/reporting v1 | Complete | Captured collector artifacts converge offline into normalized inputs; external versioned policy evaluates them with explicit severity, remediation, time-bounded exceptions, warning/failure states, preserved unknown/unavailable evidence, an explicit `as_of` date, and deterministic Markdown/JSON reports without provider re-scan or opaque scoring. |
| Project-truth reconciliation | Complete | #139 reconciled current architecture, documentation, and active backlog with merged behavior. |
| Representative operator acceptance | Complete | #137 exercised mirror/read-only/dry-run paths, exposed/fixed #141 and #143, and recorded deterministic convergence/report evidence. |

## Active sequence

### 1. Publish the v0.2.0 capability baseline (#138)

Perform release readiness, curate the changelog, reconcile README/current architecture/current plan claims, validate the exact release commit, publish through the documented immutable tag workflow, and independently download and verify the resulting assets.

Exit condition:

- the supported `v0.2.0` capability set and known limitations are explicit;
- CI, security, Nix, deep validation, and release-package gates pass at the release commit;
- published checksums and installed Linux binary are independently verified;
- the release reports its exact tag and source commit;
- a safe representative path for newly released posture convergence/reporting succeeds from the published binary.

### 2. Reassess the next concrete operator need

After the release, prioritize new work from observed operator friction, security/reliability evidence, and dependency leverage. Do not automatically broaden posture, provider mutation, ref scope, or Anthesis integration.

## Completed release-preparation sequence

### Reconcile post-v0.1 project truth (#139)

Completed before release acceptance. Current documentation/backlog distinguishes the published v0.1 surface from the post-v0.1 implementation and keeps provider/authority claims bounded.

### Exercise representative operator workflows (#137)

Completed on 2026-08-24. Live mirror/status/plan/dry-run and posture collector paths were exercised without remote mutation. Acceptance exposed the read-only-home cache defect (#141/#142) and missing convergence CLI (#143/#145); both were fixed. The final offline convergence/report path produced byte-identical repeated outputs for an explicit `--as-of` date.

## Explicit deferrals

- tags, non-default branches, wildcard refspecs, and deleted-ref reconciliation;
- tag/release mirror-drift observation beyond explicit `unknown` v1 facts;
- tag-signature/release-boundary commit posture beyond explicit `unknown` v1 facts;
- provider-derived direct-push/unreviewed commit conclusions until evidence can prove them;
- concurrent mirror mutation;
- automatic rollback or cross-repository/cross-remote transactions;
- Anthesis runtime transport/authentication and approval workflows;
- provider provisioning or hosted control-plane behavior;
- automatic provider-setting remediation from posture findings;
- automatic PR remediation from posture findings;
- scanner integration into posture policy;
- opaque numeric posture scoring;
- automatic managed-artifact mirror propagation;
- arbitrary managed file generation;
- package-manager publication beyond the repository-owned Nix flake unless separately prioritized;
- containers, release signing, and full provenance attestation;
- repository-wide performance gates without a stable workload and useful threshold.

Deferred tracks must reuse current identity, plan, policy, execution, result, evidence, test, and static-analysis substrates rather than create parallel paths.

## Simplicity constraints

- Preserve Repora as a standalone local-first controller.
- Prefer vertical capabilities over disconnected internal models.
- Keep Git-ref reconciliation, managed artifacts, routing, assessments, posture fact collection, posture policy, and optional external authorization policy as separate bounded domain contracts.
- Reuse durable `uid`, `provider:path`, shared posture fact states, and evidence concepts across posture work.
- Repository-owned observation profiles select facts only; external posture policy owns severity, expectations, remediation, and exceptions.
- External policy may add authorization but may never rewrite or weaken reviewed Repora intent.
- Keep ref-policy v1 closed and mirrors sequential until a separate decision expands them.
- Do not imply cross-remote atomicity, rollback, provider remediation, or approval semantics that are not implemented.
- Packaging may expose capability; it must not silently grant mutation authority.
- CI/Nix/posture must inspect or reuse canonical validation rather than redefine it independently.
- Hooks posture may inspect repository-owned configuration only as bounded data; it must never install, source, execute, or bootstrap target-repository hook code.
- Commit posture remains repository/process evidence only; author/committer identities, productivity metrics, blame, and intent inference are outside its contract.
- Posture convergence/policy/reporting consumes captured normalized evidence offline; it does not call providers, run scanners, or collapse findings into a numeric grade.

## Definition of done

A slice is complete only when observable behavior, appropriate failure/security tests, versioned contracts where applicable, documentation, issue acceptance criteria, independent review, and exact-head CI/security validation are complete.

For design-only slices, source behavior must remain unchanged and the implementation boundary/deferred work must be explicit.

## Maintenance

Update this plan when a capability completes, implementation order changes, or deferred work is explicitly reprioritized. GitHub issues remain authoritative for detailed acceptance criteria and live work state.