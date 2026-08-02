# ADR-0010: Versioned Domain-Specific Plan Artifacts

Status: Accepted

Decision date: 2026-08-02

Last reviewed: 2026-08-02

Supersedes: the draft universal diff abstraction previously described by ADR-0010

Implemented by: planner/executor artifact slices through PRs #42, #60, #62, #67, and the exact CLI artifact boundary under issue #8

Related issues: #3, #8, #22, #1, #6

## Context

Repora needs a reviewable, serializable, and executable boundary between planning and mutation. That boundary must preserve repository identity, exact ref preconditions, force intent, deterministic ordering, and safe recovery from stale input.

The earlier draft proposed one universal state/diff abstraction for Git refs, files, workflows, and artifacts. Repora currently has implemented experience only for Git ref reconciliation. Imposing one cross-domain model before those other domains exist would add abstraction and coupling without evidence that their identity, comparison, policy, or execution semantics are actually shared.

## Decision

Repora uses versioned, domain-specific plan artifacts.

The first accepted artifact is:

```text
Artifact {
  version: 1
  kind: repora.io/reconciliation-plan
  repositories: Repository[]
}

Repository {
  uid: durable repository identity
  id: human-facing repository identity
  actions: Action[]
}

Action {
  type: PUSH_BRANCH
  source: provider, remote name, branch
  target: provider, remote name, branch
  diff: observed target OID -> desired source OID
  force: bool
  reason: string
}
```

The exact validated artifact is both:

- exportable from `repoctl plan --artifact` for operator review;
- accepted by `repoctl apply --plan-file` without rebuilding reconciliation intent.

Convenience apply builds the same artifact internally and delegates to the same artifact execution function.

The stabilized `repora.plan` version 1 CLI response remains a compatibility projection from the exact reconciliation plans. It is not a second decision model.

## Shared conventions

Future plan domains should reuse conventions only where they are demonstrably common:

- explicit `kind` and integer `version`;
- durable repository `uid` and human-facing `id`;
- deterministic ordered serialization;
- strict parsing and unknown-field rejection;
- exclusion and redaction of credentials, resolved URLs, and unnecessary local paths;
- explicit observed and desired state;
- explicit destructive or force intent;
- validation before side effects;
- stale-input rejection and re-planning recovery.

These conventions do not require one universal state object or action vocabulary.

## Rejected universal abstraction

Repora does not currently adopt this generic model:

```text
StateObject {
  domain: refs | files | workflows | artifacts
  identity: string
  desired: any
  observed: any
  diff: structured delta
  state: EQUAL | DRIFT | DIVERGED | UNKNOWN
}
```

It also does not require all domains to normalize into generic `SYNC`, `UPDATE`, `CREATE`, or `DELETE` actions.

The model may be reconsidered only after at least two implemented domains demonstrate shared semantics that reduce code and policy complexity rather than merely sharing field names.

## Execution boundary

The executor validates the artifact, converts it to the internal ref-action model, validates every action, and re-resolves every expected source and target OID before action zero mutates a remote.

Invalid artifacts, unsupported repository cardinality for an executor invocation, topology mismatches, missing force authorization, and stale refs fail closed.

A force-with-lease push is defense in depth after complete stale-ref preflight. It does not authorize replay of an old artifact.

Multi-repository orchestration may partition one validated artifact into single-repository executor calls. There is no cross-repository transaction or implied atomicity.

## Identity and safety

The artifact uses durable repository `uid` for configuration matching. Human-facing `id` may change without redefining durable identity.

The artifact excludes:

- runtime transport URLs;
- credentials and authorization material;
- raw command lines and environments;
- local cache or checkout paths.

Validation rejects malformed versions, kinds, identifiers, symbolic refs, OIDs, action types, unknown fields, and unsafe serialized values.

## Consequences

### Positive

- The reviewed plan can be the exact executor input.
- Planner and executor can be tested independently.
- Stale and destructive operations remain explicit.
- The current Git-ref model stays small and understandable.
- Other domains can evolve without being forced into ref-specific semantics.
- Compatibility output can remain stable as a projection.

### Costs

- Each new mutation domain needs its own action and schema design.
- Shared tooling must dispatch on artifact kind or tagged action type.
- Compatibility and schema evolution remain deliberate maintenance work.
- Operators must re-plan when current refs no longer match the artifact.

## Future decision rule

A shared cross-domain abstraction requires a later ADR with evidence from implemented consumers. Similar vocabulary alone is insufficient; the abstraction must demonstrate simpler validation, policy, execution, and recovery behavior across those domains.
