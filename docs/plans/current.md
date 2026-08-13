# Current implementation plan

Status: Active

## Current objective

Repora's v0.1 mirror-controller release and the first managed-artifact/routing/assessment slices are complete. The current objective is to keep the standalone controller easy to compose while avoiding new runtime authority without an explicit reviewed contract.

## Completed foundation

| Area | State | Notes |
| --- | --- | --- |
| Durable `uid` identity | Complete | Cache and durable artifacts use logical identity. |
| Provider/path topology | Complete | Multiple mirrors use stable provider/path identity. |
| Ref policy v1 | Complete | Default branch only; destructive actions require explicit authorization. |
| Multi-mirror status/planning | Complete | Exact artifact v2 is provider/path-bound and deterministic. |
| Stale-safe execution | Complete | Complete OID preflight, exact leases, partial-result evidence, and no implicit rollback. |
| Execution evidence | Complete | Immutable INTENT/RESULT journals preserve reviewed intent and outcomes. |
| v0.1 release | Complete | Protected tag and published assets were independently verified. |
| Managed README lifecycle | Complete | Separate deterministic plan/apply domain with exact stale preflight, leased canonical push, and durable evidence. |
| Document routing foundation | Complete | Deterministic route tests, trust tiers, manifests, context receipts, hierarchy/AST routing slices. |
| Repository assessment | Complete | Evidence-backed assessment/scorecard contracts are implemented. |
| Managed-artifact Go contract consolidation | Complete | `internal/managedartifact` is the single authoritative plan-v1 implementation. |

## Active sequence

### 1. Finish optional Anthesis policy integration design (#30)

Exit condition:

- an accepted ADR defines the optional additive policy boundary;
- `pre_apply` is selected as the first authorization seam;
- deterministic facts/decision/evidence contracts and fail-closed `enforce` semantics are documented;
- Repora standalone operation remains the default;
- runtime Anthesis transport and policy-engine coupling remain explicitly deferred.

### 2. Add standalone Nix flake packaging (#115)

Repora should expose a repository-owned Nix package/application/check contract suitable for independent use and Dubnium composition without depending on private Dubnium modules or granting repository mutation authority through packaging.

Exit condition:

- root flake exports the canonical package/app/checks/dev-shell/formatter surface;
- package and checks reuse existing Go dependency/build/test/static-analysis contracts;
- repository-mutation tests remain disposable and hermetic;
- standalone Nix usage is documented and requires no Dubnium access.

### 3. Reassess runtime policy integration

After #30 design and #115 packaging, do not automatically implement Anthesis runtime coupling. Resume only when there is a concrete evaluator contract/deployment path and a clear operator need.

If resumed, the first runtime slice must remain transport-free and mutation-neutral: versioned policy facts/decision schemas, deterministic facts construction, evaluator interface/fakes, and local policy evidence before inserting a real gate.

## Explicit deferrals

- tags, non-default branches, wildcard refspecs, and deleted-ref reconciliation;
- concurrent mirror mutation;
- automatic rollback or cross-repository/cross-remote transactions;
- Anthesis runtime transport/authentication and approval workflows;
- provider provisioning or hosted control-plane behavior;
- package-manager publication beyond the repository-owned Nix flake unless separately prioritized;
- containers, release signing, and full provenance attestation;
- repository-wide performance gates without a stable workload and useful threshold.

Deferred tracks must reuse the current plan, policy, execution, result, and evidence substrate rather than create parallel paths.

## Simplicity constraints

- Preserve Repora as a standalone local-first controller.
- Prefer vertical capabilities over disconnected internal models.
- Keep Git-ref reconciliation, managed artifacts, routing, assessments, and policy integration as separate domain contracts.
- External policy may add authorization but may never rewrite or weaken reviewed Repora intent.
- Keep ref-policy v1 closed and mirrors sequential until a separate decision expands them.
- Do not imply cross-remote atomicity, rollback, or approval semantics that are not implemented.
- Packaging may expose capability; it must not silently grant mutation authority.

## Definition of done

A slice is complete only when observable behavior, failure/recovery tests, versioned contracts where applicable, documentation, issue acceptance criteria, independent review, and CI/security validation are complete.

For design-only slices, source behavior must remain unchanged and the implementation boundary/deferred work must be explicit.

## Maintenance

Update this plan when a capability completes, implementation order changes, or deferred work is explicitly reprioritized.
