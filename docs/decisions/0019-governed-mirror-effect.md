# ADR-0019: Govern mirror effects through a typed Capability Gateway boundary

- Status: **Proposed** — no runtime authority or schema is accepted by this record
- Date: 2026-10-11
- Related: #211, #212, #213; #30 / ADR-0018
- Supersedes: none
- Implementation: not started; consult `docs/architecture/governed-mirror-reconciliation.md` for the proposed detailed contract

## Context

Repora already performs deterministic default-branch mirror observation, reviewed plans, local ref-policy checks, complete preflight, leased pushes, and immutable INTENT/RESULT evidence. ADR-0018 selected an optional additive Anthesis `pre_apply` authorization seam, but deliberately deferred runtime transport/authentication.

Dubnium's Capability Gateway is the project-wide governed-effect admission boundary for less-trusted callers. Simply connecting an agent to a credential-bearing `repoctl apply` or exposing a generic remote Git command would allow unreviewed operations and bypass authenticated authorization, regardless of policy-response quality.

## Proposed decision

Use exactly one versioned, typed effect — provisionally `repository.mirror.reconcile` — admitted by a sealed Capability Gateway client and dispatched to a **Repora-owned trusted domain effector** that reuses existing reconciliation semantics.

Repora performs trusted prepare: validate exact reviewed plan bytes/identity and invocation intent, bind deterministic facts to the concrete effect, and reject unsupported/stale/ambiguous state. Gateway binds trustworthy actor context and durable idempotent request identity; Anthesis evaluates the exact prepared facts; only a fresh, persisted explicit `allow` can enter the governed Repora execution boundary. The effector independently rechecks exact decision/actor/manifest binding and mutation-critical state before durable Git INTENT and any push.

A successful policy decision cannot rewrite plans, infer `--force`, broaden refs, replace leased-force semantics, authorize another action, or bypass Repora's local constraints. All provider write credentials remain in a principal unavailable to agents and remote requestors.

Trusted standalone `repoctl` retains its existing behavior when integration is disabled. An environment claiming **governed** execution must enforce complete mediation outside the request JSON and must not offer a direct agent Git/CLI credential bypass.

## Rejected alternatives

1. **Anthesis client embedded in every untrusted `repoctl` process:** useful for a trusted local CLI, but insufficient to authenticate remote principals or prevent direct bypass when the process holds canonical write credentials.
2. **Generic Gateway-to-shell/Git proxy:** broadens target, argument, environment and credential authority beyond the reviewed operation.
3. **Reimplement mirror reconciliation inside the Gateway:** duplicates domain rules and invalidates the existing plan and failure evidence model.
4. **Make Gateway mandatory for all local-first operations:** breaks offline/trusted CLI utility without improving governed deployment complete mediation.

## Consequences

- Adds a narrow trusted provider/client integration and independent Gateway/Anthesis deployment prerequisites; Repora's existing engine stays authoritative.
- The transport principal and effective actor must be separately authenticated and bound. Trust-domain labels and caller-supplied actor strings never grant authority.
- A repeated idempotent request returns the existing effect status; ambiguous or partial execution requires fresh observation/review rather than replay.
- Gateway admission, Anthesis evidence, Repora INTENT and RESULT are correlated; receipt persistence failure is not interpreted as rollback.
- Mirror v1 remains default-branch-only, with sequential independent target pushes and honest partial outcomes.
- This proposal makes **no claim** of a functioning runtime Capability Gateway effect, approval flow, or production actor delegation.

## Compatibility and implementation boundary

No current `repora.yaml`, plan artifact v1/v2, execution-record, apply v2/v3, or trusted CLI contract changes solely because this ADR is written. An implementation requires a versioned effect request/facts/receipt contract, exact canonical digests, trusted preparation and actor provenance, idempotency, scoped credentials, fail-closed evidence, disposable-Git regression tests, independent review, and explicitly approved activation.

No new capability name is considered registered merely because it appears in this design. Prove it in the authoritative Gateway registry first. Do not add projection, promotion, managed README or CI administration as aliases of a mirror effect.

## Validation gates

Before changing this ADR from Proposed to Accepted, #212 must resolve the Gateway preparation/authorization lifecycle, shared schema ownership, trustworthy actor delegation, and whether Anthesis's effect-time authorization can be verified by the actual effector. Before classifying the behavior Implemented, #213 must prove a production-qualified Gateway path, no-bypass credential isolation and exact-head negative/partial-failure evidence.

See `docs/architecture/governed-mirror-reconciliation.md` and `docs/architecture/end-to-end-acceptance.md`.
