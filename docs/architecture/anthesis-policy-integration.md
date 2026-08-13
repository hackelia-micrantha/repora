# Deterministic Anthesis policy integration

Status: Proposed

Issue: #30

## Purpose

This document defines an optional deterministic policy seam between Repora's reviewed Git-ref reconciliation intent and Anthesis policy evaluation.

The integration is deliberately narrow:

- Repora owns repository observation, exact reconciliation planning, local ref policy, command authorization, Git execution, results, and execution journals.
- Anthesis may evaluate structured facts about an already reviewed Repora effect and return a deterministic policy decision.
- Anthesis does not build, edit, reorder, retarget, or execute Repora plans.
- Repora remains fully usable when Anthesis integration is disabled.

This is a design contract only. It does not make Anthesis a runtime dependency.

## First integration point

The first integration point, if implemented, is `pre_apply`.

It occurs after Repora has:

1. loaded or built the exact reconciliation artifact;
2. rebound durable provider/path identities to current runtime aliases;
3. validated repository topology, ref-policy agreement, default branches, and reviewed force intent;
4. required existing command-level `--force` authorization when the artifact contains destructive actions.

It occurs before Repora:

1. persists the Git execution-record INTENT;
2. performs final expected-OID executor preflight;
3. executes any push.

```text
exact reviewed reconciliation artifact
  -> Repora topology/policy/force preparation
  -> build immutable policy facts
  -> optional Anthesis evaluation
  -> persist policy evaluation evidence
  -> enforce decision
       blocked -> stop before Git execution INTENT/mutation
       allowed -> existing execution INTENT -> OID preflight -> pushes -> RESULT
```

This placement gives Anthesis concrete reviewed effects to authorize without granting it authority over Git planning or mutation.

### Why not the other candidate seams first

- `pre_plan` has too little concrete effect information and risks turning policy into target discovery or credential orchestration.
- `post_plan` is close to the chosen seam but does not yet include final local topology/force authorization bindings.
- `post_apply` and `post_failure` are evidence hooks, not authorization gates.

Future evidence-only hooks may reuse the same contract family, but they are not required for the first implementation.

## Non-bypassable local controls

Anthesis policy is additive. It cannot weaken Repora's built-in controls.

In particular:

- default-branch-only ref-policy remains authoritative;
- reviewed force intent remains required for ahead/diverged mirrors;
- command-level `--force` remains required for destructive reviewed actions;
- complete expected-OID preflight remains required before action zero;
- force-with-lease remains required for forced pushes;
- stale artifacts still require fresh observation and a new plan.

An Anthesis `allow` never converts a locally invalid or unauthorized Repora operation into a valid one.

## Policy configuration boundary

A future implementation should use explicit optional configuration. Conceptually:

```yaml
policy_integration:
  anthesis:
    mode: disabled | record_only | warn | enforce
    endpoint: local-runtime-reference
    timeout: bounded-duration
```

The exact configuration shape is deferred to implementation review.

Rules:

- `disabled` is the standalone default.
- No endpoint, token, credential, or transport URL is serialized into Repora plan artifacts or policy facts.
- Credentials remain runtime-only configuration and must not enter durable evidence.
- Enabling Anthesis must be an explicit operator choice.

## Enforcement modes

### `disabled`

Repora does not call Anthesis. Existing standalone behavior is unchanged.

### `record_only`

Repora may evaluate policy and persist the evaluation, but the policy result never blocks execution. Existing Repora local controls still apply.

Evaluation transport/schema failures are reported as policy-evidence failures but do not independently deny the operation. An implementation must make this non-enforcing status explicit in output/evidence.

### `warn`

`deny`, `require_approval`, `warn`, or policy-evaluation failures are surfaced prominently and persisted, but do not independently block execution. Existing Repora local controls still apply.

This mode is observational and must never be described as authorization enforcement.

### `enforce`

Only a valid explicit `allow` decision authorizes progression to the existing Git execution boundary.

All other outcomes block before Git execution INTENT and before mutation:

- `deny`;
- `warn`;
- `require_approval`;
- `record`;
- `not_applicable`;
- unavailable evaluator;
- timeout;
- malformed response;
- unsupported contract/policy version;
- ambiguous response;
- policy-evidence persistence failure.

This is the fail-closed mode.

## Decision contract

A future versioned response contract should contain only bounded deterministic fields such as:

```json
{
  "kind": "repora.io/policy-decision",
  "version": 1,
  "integration_point": "pre_apply",
  "artifact_sha256": "<reviewed-artifact-digest>",
  "facts_sha256": "<canonical-facts-digest>",
  "decision": "allow",
  "policy": {
    "id": "mirror-destructive-policy",
    "version": "1"
  },
  "reason_code": "approved_by_policy"
}
```

Allowed decision values are:

- `allow` — policy authorizes continuing, subject to all Repora controls;
- `deny` — policy rejects the reviewed effect;
- `warn` — non-authorizing advisory result;
- `require_approval` — an external approval is required; Repora does not fabricate or satisfy it;
- `record` — evidence-only classification;
- `not_applicable` — no applicable rule produced authorization.

Free-form policy prose should not be execution authority. A bounded `reason_code` or similarly controlled explanation may be retained for audit display.

A response must bind to the exact artifact and exact facts digest. A response for a different digest is invalid.

## Policy facts v1

The first facts contract should describe the exact reviewed effect without transport secrets or mutable aliases.

Conceptual shape:

```json
{
  "kind": "repora.io/policy-facts",
  "version": 1,
  "integration_point": "pre_apply",
  "operation": "mirror_reconciliation",
  "artifact_sha256": "<sha256>",
  "actor": {
    "kind": "human",
    "id": null
  },
  "repository": {
    "uid": "repo.repora",
    "canonical": {
      "provider": "gitlab",
      "path": "micrantha/repora",
      "branch": "main"
    }
  },
  "actions": [
    {
      "target": {
        "provider": "github",
        "path": "hackelia-micrantha/repora",
        "branch": "main"
      },
      "before_oid": "<reviewed-target-oid>",
      "desired_oid": "<reviewed-canonical-oid>",
      "force": false,
      "reason": "behind"
    }
  ],
  "risk": {
    "destructive": false,
    "action_count": 1
  }
}
```

### Stable inputs

Facts should be derived from:

- exact reconciliation artifact kind/version/digest;
- durable repository UID;
- canonical provider/path/default branch;
- target provider/path/default branch;
- reviewed before/desired OIDs;
- reviewed force intent and planner reason;
- command actor classification supplied by the runtime boundary;
- bounded derived risk flags.

Facts must not contain:

- HTTPS/SSH remote URLs;
- credentials, tokens, credential-helper output, headers, environment values, or command lines;
- local cache/worktree paths;
- runtime Git aliases as authority;
- arbitrary repository file content;
- mutable timestamps inside the canonical digest body.

### Actor

Actor identity distinguishes:

- `human`;
- `automation`;
- `service`.

Actor `id` is optional unless a future integration can supply a trustworthy identity. Repora must not infer a security principal from an untrusted environment string and then treat it as authorization evidence.

## Determinism and canonicalization

Policy facts are evidence and authorization input, so equivalent reviewed intent must produce equivalent canonical facts.

A future implementation must define canonical serialization with:

- explicit kind/version;
- stable repository/action ordering inherited from the exact artifact;
- no timestamps or random IDs in the canonical digest body;
- normalized bounded strings;
- SHA-256 over the canonical serialized facts;
- exact reviewed artifact SHA-256 included in facts and response binding.

Runtime timestamps may exist in outer evidence records but are not part of authorization identity.

## Evidence

Policy evaluation evidence is distinct from the existing Git execution-record contract.

A future policy-evaluation record should retain enough information to explain a decision without becoming replay authority:

- record kind/version;
- execution or evaluation ID as evidence metadata;
- integration point;
- enforcement mode;
- artifact SHA-256;
- facts SHA-256;
- canonical policy facts or a bounded facts reference;
- decision;
- policy ID/version when supplied;
- bounded reason code;
- evaluator outcome (`evaluated`, `unavailable`, `timeout`, `invalid`);
- final gate outcome (`allowed`, `blocked`, `advisory`);
- timestamp as evidence metadata.

Policy evidence must be persisted before an enforcing `allow` can progress to Git execution INTENT. In `enforce` mode, failure to persist policy evidence blocks execution.

Existing Git execution journals remain the authoritative evidence of attempted Git mutation. Policy evidence explains the authorization gate; it does not replace execution INTENT/RESULT.

## Replay and challenge

A policy decision can be challenged without treating it as reusable authorization:

1. obtain the exact reconciliation artifact;
2. recompute canonical policy facts from that artifact and the recorded actor/risk inputs;
3. verify `artifact_sha256` and `facts_sha256`;
4. identify the policy ID/version recorded by Anthesis;
5. re-evaluate for comparison when that policy version is available.

A previous `allow` must never be replayed to authorize a new artifact, changed facts, changed repository state, or changed actor.

## Failure behavior

| Condition | disabled | record_only / warn | enforce |
| --- | --- | --- | --- |
| Anthesis unavailable | no effect | surface + continue under Repora controls | block |
| Timeout | no effect | surface + continue under Repora controls | block |
| Invalid/malformed response | no effect | surface + continue under Repora controls | block |
| Unsupported contract/policy version | no effect | surface + continue under Repora controls | block |
| Digest mismatch | no effect | surface + continue under Repora controls | block |
| `deny` | no effect | advisory only | block |
| `require_approval` | no effect | advisory only | block |
| `not_applicable` | no effect | advisory only | block |
| Policy evidence write fails | no effect | return/report evidence failure per mode contract | block |
| Explicit valid `allow` | no effect | record/advisory | continue to existing execution boundary |

In all modes, Repora's own invalid topology, missing force authorization, stale refs, or executor preflight failure still block independently.

## Transport and privilege boundary

The first implementation should use a small evaluator interface owned by Repora rather than embedding Anthesis policy logic.

Conceptually:

```go
type PolicyEvaluator interface {
    Evaluate(ctx context.Context, facts []byte) ([]byte, error)
}
```

The transport implementation may later call a local process, Unix socket, localhost service, or remote service, but transport selection is not part of this design.

The evaluator receives facts only. It receives no Git client, repository credentials, filesystem handle, plan mutation capability, or push capability.

## Compatibility

- Existing reconciliation artifact v1/v2 semantics do not change.
- Existing `repora.apply` and execution-record contracts do not gain implicit Anthesis meaning.
- Existing standalone commands remain valid with integration disabled.
- Enabling policy must not change artifact identity or rewrite reviewed plans.
- A future policy-facts or policy-decision schema is versioned independently from reconciliation artifacts.

## Implementation boundary

Issue #30 is satisfied by this design. Runtime implementation remains explicitly deferred until there is a concrete Anthesis evaluator contract and deployment path worth supporting.

When implementation is resumed, the first slice should be deliberately small:

1. define versioned `policy-facts` and `policy-decision` JSON schemas;
2. add deterministic facts construction from exact reconciliation artifact v2;
3. add a transport-free `PolicyEvaluator` interface plus fake evaluator tests;
4. add local policy-evaluation evidence with no Git mutation changes;
5. only then insert the optional `pre_apply` gate before Git execution INTENT;
6. separately decide any real Anthesis transport/authentication adapter.

No runtime Anthesis transport, approval workflow, or policy language is approved by this design.
