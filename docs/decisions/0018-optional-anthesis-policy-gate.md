# ADR-0018: Optional deterministic Anthesis policy gate

Status: Accepted

Decision date: 2026-08-13

Review date: 2026-08-13

Related issue: #30

## Context

Repora already has a closed local ref policy, exact reviewed reconciliation artifacts, command-level destructive authorization, fail-closed execution evidence, complete expected-OID preflight, and force-with-lease execution.

A future Anthesis integration could add organization-specific deterministic authorization for reviewed effects, but careless integration would risk:

- making Anthesis a mandatory runtime dependency;
- letting an external policy service alter Git intent;
- weakening Repora's local force/stale controls;
- introducing credentials or transport details into durable artifacts;
- making policy decisions reusable as mutation authority;
- coupling Repora to the Anthesis policy language or deployment topology.

## Decision

Anthesis integration is optional and additive.

If implemented, the first authorization seam is `pre_apply`, after Repora has bound and locally validated the exact reviewed reconciliation artifact and required existing force authorization, but before Git execution INTENT and before any push.

Repora supplies canonical structured policy facts derived from the exact reviewed artifact. Anthesis returns a separate versioned decision bound to the artifact digest and facts digest.

Anthesis does not build, modify, reorder, retarget, or execute Repora plans.

Repora's local controls remain independently mandatory. An Anthesis `allow` cannot authorize an operation that Repora itself rejects.

Standalone behavior remains the default. `disabled` is the default integration mode.

In future `enforce` mode, only an explicit valid `allow` may proceed. Unavailability, timeout, malformed/ambiguous responses, unsupported versions, digest mismatch, `deny`, `warn`, `record`, `require_approval`, `not_applicable`, or required policy-evidence persistence failure all fail closed before Git execution INTENT and mutation.

Policy evaluation evidence is separate from Git execution INTENT/RESULT. A previous policy decision is evidence, not replay authority.

## Alternatives rejected

### Make Anthesis mandatory

Rejected because Repora must remain useful as a standalone local-first mirror controller.

### Let Anthesis produce or rewrite reconciliation plans

Rejected because Git planning and target identity belong to Repora. External plan mutation would break exact review, stale detection, and authority separation.

### Gate before planning

Rejected as the first seam because pre-plan policy lacks the concrete reviewed effect and risks turning policy into discovery/credential orchestration.

### Evaluate after Git execution INTENT

Rejected for the first design because denied policy would create Git execution intent for an operation that was never authorized to enter the Git mutation boundary. Policy evaluation receives its own evidence record instead.

### Reuse policy `allow` as approval/replay authority

Rejected because authorization must bind to exact artifact/facts/current execution context. A prior allow cannot authorize changed state or intent.

## Consequences

Positive:

- Anthesis can add policy without becoming Git authority.
- Existing Repora safety properties remain intact.
- Policy decisions are attributable to exact reviewed effects.
- Standalone operation remains unchanged when disabled.
- Transport/authentication can evolve independently from the facts/decision contract.

Costs:

- policy facts, decisions, and evidence require separate versioned contracts;
- enforcing operation adds a synchronous external dependency when explicitly enabled;
- denied/advisory policy evaluations require a separate evidence lifecycle from Git execution journals;
- approval workflows are not solved by this ADR.

## Security implications

- The evaluator receives only bounded policy facts, not Git credentials or mutation capabilities.
- Durable facts exclude transport URLs, credentials, environment secrets, local paths, and command lines.
- In enforce mode, explicit allow is required and failures are fail closed.
- Existing `--force`, OID preflight, and force-with-lease remain mandatory.
- Decision binding uses exact artifact/facts digests.
- Policy evidence must be durable before an enforcing allow enters Git execution.

## Compatibility

No existing reconciliation artifact, apply result, execution-record, CLI, or configuration contract is changed by this accepted design.

Future policy contracts are versioned independently. Existing runtime behavior remains unchanged until a separate implementation PR is accepted.

## Implementation boundary

Accepted now:

- optional additive policy model;
- `pre_apply` as first seam;
- deterministic digest-bound facts/decision contracts;
- fail-closed enforce semantics;
- separate policy evidence;
- standalone default.

Deferred:

- JSON schemas and Go types;
- evaluator interface implementation;
- CLI/configuration syntax;
- Anthesis transport/authentication;
- approval workflow;
- runtime policy gate insertion.

The detailed proposed contract is documented in `docs/architecture/anthesis-policy-integration.md`.

## Validation

This ADR is a design decision only. Implementation validation will require deterministic schema/serialization tests, failure-mode tests, evidence tests, and proof that disabled mode preserves current behavior and that an evaluator cannot mutate Git intent.
