# End-to-end acceptance contract

Status: **Proposed** — product acceptance design; current capabilities are identified explicitly.
Owners: Repora (#211); implementation and operational evidence remain in owning issues.
Drafted: 2026-10-11

## Purpose and scope

"Done" is scoped to a **named released capability** or **declared milestone**, not an ever-expanding promise that every repository lifecycle operation has shipped. Repora remains a standalone local-first Git controller. Gateway-governed automation is an additional deployment profile, not a prerequisite for invoking a trusted local CLI.

The product-level invariant is:

> Given declared, strictly validated topology and observed state, Repora generates an exact reviewed plan, performs only the planned bounded effect under mandatory local guards and (for governed callers) exact external authorization, then records and verifies the outcome without concealing partial success, uncertainty, or recovery limitations.

A source tree, passing test suite, merged PR, policy `allow`, or released archive alone is **not** an end-to-end acceptance result.

## Capability boundaries

| Area | Current verified implementation scope | Product acceptance expansion |
| --- | --- | --- |
| Mirror v1 | GitLab canonical; GitHub/GitLab/Bitbucket Cloud mirrors; default branch; multi-target exact plan, force-with-lease, stale preflight, INTENT/RESULT and partial outcomes | Outstanding installed-release operator qualification #221 (historical release issue #138); governed execution #212/#213 |
| Managed README | Opt-in root README, deterministic exact plan and leased canonical push; mirror propagation is separate fresh plan | Keep separate authorization domain; do not silently gain authority from mirror allow |
| Posture | Read-only facts, explicit observed/unknown/unavailable, deterministic offline reporting | Operator evidence and remediation planning never imply mutation |
| Rich topology | #149/#150 proposal: roles, trust domains, projections, promotions, archives | Runtime acceptance is per relationship type, not "one generic sync" |
| Forgejo | #151/#152 open | Qualified instance+canonical behavior before advertising |
| CI/CD management | #160/#184–190 open | Separate resource-specific plans and effect authority; no generic CI compiler |
| Capability Gateway | External Dubnium experimental infrastructure, not an implemented Repora integration | Qualify exact governed mirror effect #212/#213 |
| Recovery archive | #156 open | Git bundle/ref coverage and isolated restore; forge state explicitly separately scoped |

Do not equate default-branch reconciliation with `git push --mirror`: tags, wildcard refs, deleted refs, LFS, repository settings, issues, PRs, Actions and forge database state are not covered by mirror v1. Expand scope only via a new versioned ref/relationship contract and separate acceptance (#215).

## Gate A — published standalone mirror controller

The v0.2.0 publication and download verification are recorded in #138 (which is closed, despite remaining unchecked operator-smoke criteria). **Gate A remains unqualified** until the remaining read-only operator checks are evidenced under #221 and reconciled into #211. The tests must exercise the published `v0.2.0` Linux binary, not only current source or a different installed toolchain:

1. Verify archive checksum, embedded release tag and exact commit (already evidenced in #138).
2. Against an accessible real GitLab canonical plus supported mirror(s), run `repoctl status -f CONFIG`, `repoctl plan -f CONFIG` and `repoctl apply -f CONFIG --dry-run`. Record actual exit status, identity topology, auth limitations and redacted output. **No live provider mutation is required or authorized for release smoke.**
3. Run installed-binary `repoctl posture converge` and `repoctl posture report` with valid captured normalized inputs. Record output digest and the expected observed/unknown/unavailable semantics.
4. Confirm existing integration/CLI E2E fixtures cover `EQUAL`, `BEHIND`, `AHEAD`, `DIVERGED`, changed target OID before action zero, explicit destructive intent, force-with-lease failure, and a mixed `APPLIED/FAILED/APPLIED` result. A fixture must not claim remote production access.
5. Record exact release binaries/tests/inputs and their limitations. Independent package verification is distinct from native macOS/Windows runtime verification.

Passing Gate A qualifies the **specified default-branch product slice**, not Gateway automation or full-ref recovery.

## Gate B — governed mirror automation

Depends on accepted design #212 and implementation #213 plus authoritative Gateway prerequisites.

Use one immutable reviewed plan and one sealed effect `repository.mirror.reconcile`. The governed deployment must prove:

- A proposing agent/scheduler/front door has **no canonical/mirror write credential** and no alternate direct path that bypasses the Gateway.
- Trusted preparation validates exact plan bytes/digest, repository UID, provider/instance/path, target refs/OIDs, force intent and applicable local policy before authorization.
- The Gateway binds authenticated transport principal and any separately verified effective actor provenance. A request-body `actor` cannot self-authorize.
- Anthesis authorizes the exact effect; only valid explicit `allow` is enforcing. Denial, approval-required, unavailable, timeout, malformed or mismatched result fail before Git execution INTENT and **before mutation**.
- Effect-time local preflight checks required OIDs and authorization-relevant state. Changed dependencies cannot reuse old allow or plan. Leased force push is mandatory when destructive.
- For a multi-mirror request, revocation/policy dependency freshness is checked before **each** sequential push; if not implementable, qualify only single-mirror governed requests and leave multi-mirror governed acceptance open.
- Provider effect executes via the **same Repora domain logic** as trusted standalone apply, with no generic Gateway shell or provider proxy.
- Repora's immutable INTENT/RESULT and target outcome evidence can be correlated to Gateway invocation/decision, plan digest and bounded actor context without serializing credentials.
- Duplicate request returns an existing operation status, not another write. An interrupted/ambiguous or partial operation does not authorize replay: inspect/re-observe and review a new plan.
- A disposable Git integration suite exercises both positive paths and negative failure cases, including evidence persistence failure, unexpected target, forged actor, approval replay, stale branch, target redirect/credential leak, and one failing mirror.
- A real operator-visible workflow is exercised: Dubnium scheduler (or equivalent external trigger) observes canonical drift, Repora produces a reviewable exact plan, policy/approval admits only that effect, the effector reconciles an eligible target, a fresh observation verifies convergence, and the operator receives a correlated report/receipt. Scheduling and alert delivery remain external service responsibilities. A destructive request requires separate explicit intent; a scheduled read cannot turn into an unattended force push.

A missing production Gateway or actor-binding prerequisite means Gate B is **blocked**, not "passed by fixture". A local no-effect fixture is useful implementation evidence but not operator acceptance.

## Gate C — controlled multi-provider lifecycle (independent workstream)

This gate is **independent of and not a blocker for** Gate A or B. Relationship-specific deployment may reuse verified Gateway governance but must not inherit mirror authorization.

- #149/#150: project/repository/endpoint/relationship model, compatibility and negative trust-role tests.
- #151/#152: provider-neutral canonical and generic Forgejo instances.
- #153: private-to-public clean **projection** with exact input/profile/materialized-tree identities and public disclosure verification. No private history by default.
- #154: exact **promotion** from staging to canonical with its own authority (not borrowed from mirror allow).
- #155: public PR/commit import creates candidate proposals only; any canonical mutation requires separate private review.
- #156: archive manifest and isolated restore with explicit ref, object, LFS and forge-data coverage.
- #160/#184–190: CI policy/provider attachments, stale-safe narrow provider management and second-provider proof. Provider CI remains execution authority for workflows.
- Backups, scheduling, hosting, secrets, and runtime runners remain Dubnium/provider responsibilities.

## Evidence bundle for any claim of "done"

For each supported operation retain: repository UID and provider/path/instance; exact release/build SHA; topology/ref-policy/profile version or digest; captured observed state; reviewed plan bytes/digest; actor/admission/approval evidence where required; execution INTENT; per-target RESULT; independent post-action observation; negative-test matrix; security and compatibility validation; known unsupported state; operator recovery steps.

Evidence states are **pass, fail, partial, unknown, unavailable, blocked** as applicable. Never convert missing data or unverified accepted provider mutation into success. A successful push is not cross-remote atomicity; a failed RESULT write does not undo an already accepted provider effect.

## Milestone validation and closure

- Gate A: record complete installed-binary operator smoke under #221 and reconcile #211, including the historical unchecked acceptance criteria in closed #138. A closed release issue is not evidence that the smoke passed.
- Gate B: close #212 with a reviewed proposed/accepted architecture contract; close #213 only after exact-head tests, independent review, actual governed-domain qualification and bypass analysis.
- Gate C: close each separate workstream only on its own contracts/evidence. Do not close #149 or #160 because an umbrella checklist exists.
- No host deployment, provider mutation or credential exposure is implied by documentation, issue closure, a merge, or a Gateway `allow`.

## Related

- `docs/architecture/governed-mirror-reconciliation.md`
- `docs/architecture/anthesis-policy-integration.md` and ADR-0018
- `docs/architecture/repository-topology-and-trust-domains.md`
- `docs/plans/current.md`, #138, #149, #160, #211–213, #215, #221
