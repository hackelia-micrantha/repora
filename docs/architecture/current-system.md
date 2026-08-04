# Current system architecture

Status: Current

This document describes merged Repora behavior. Future architecture belongs in proposals or ADRs until implemented.

## Product boundary

Repora is a local-first Git mirror controller exposed through the `repoctl` Go CLI.

For each repository entry it supports:

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

It does not provide tags, non-default branches, deleted-ref reconciliation, provider provisioning, rollback, cross-remote transactions, or a hosted control plane.

## Runtime flow

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

## Package ownership

| Package | Owns | Must not own |
| --- | --- | --- |
| `internal/config` | strict YAML, durable identity, safe endpoint paths, topology/ref-policy normalization, duplicate target rejection | Runtime URL derivation or Git operations |
| `internal/refpolicy` | closed versioned ref scope and relationship-to-intent decisions | Git operations or command authorization |
| `internal/transport` | runtime provider/path URL resolution | Durable identity or policy |
| `internal/status` | canonical/mirror observation, target identity, divergence, and commit evidence | Mutation decisions or pushes |
| `internal/plan` | deterministic reconciliation actions and compatibility projection | Git reads/writes or durable serialization |
| `internal/planartifact` | versioned exact artifact parsing, provider-path validation, historical compatibility, and plan conversion | Observation or execution policy |
| `internal/executor` | complete OID preflight, runtime bindings, sequential independent pushes, leases, and action outcomes | Recomputing status or policy |
| `internal/apply` | artifact construction, topology/status/policy binding, force authorization, audit orchestration, and apply v2/v3 results | Implicit target selection or rollback policy |
| `internal/journal` | immutable path-bound intent/result evidence, digest correlation, redaction, and local persistence | Mutation or replay authority |
| `internal/git` | bounded Git subprocesses, cache safety, refs, pushes, leases, timeouts, and redaction | Product policy or identity |
| `cmd/repoctl` | command routing, preparation aggregation, concurrency, output versions, artifact I/O, and exit semantics | Git mechanics or duplicated planning |

## Identity and runtime binding

Repora distinguishes:

- `id`: human-facing repository alias;
- `uid`: durable logical repository identity;
- `(provider, path)`: durable repository/mirror selector;
- configuration index: deterministic order only;
- resolved URL and Git remote alias: ephemeral runtime state.

Status v2, artifact v2, apply v3, and execution-record v3 use provider/path identity. Imported targets bind to current local aliases through a separate runtime map. The artifact and its digest are never rewritten, so mirror reordering cannot retarget reviewed intent.

## Reference policy and authorization

Ref-policy v1 supports exactly:

- `scope: default-branch-only`;
- `destructive: require-force`.

Planning records forced intent for ahead or diverged mirrors. A real command containing any forced action requires `--force` before journal creation or mutation. The flag authorizes only actions already marked forced. Dry-run validates forced actions without authorization.

## Preparation and preflight

Before any selected repository mutates, Repora completes observation and exact-artifact preparation for every selected repository. Artifact version, repository cardinality, and serialization are validated. If any preparation fails, no intent is written and no repository mutates.

For each repository, before intent persistence Repora validates:

- durable UID;
- canonical provider/path;
- every configured mirror target exactly once;
- complete current status;
- state/action/force agreement under policy v1;
- current default branches through runtime bindings.

After fail-closed intent persistence, executor preflight validates every expected source and target OID before action zero. Any stale action prevents every push for that repository, marks the offending action `STALE`, leaves unattempted actions `SKIPPED`, and still attempts result persistence.

## Independent mutation semantics

After complete repository preflight:

- actions execute sequentially in exact artifact order;
- normal actions use normal push;
- forced actions use the reviewed expected target OID as force-with-lease;
- success becomes `APPLIED` with `after = desired`;
- runtime failure becomes `FAILED`;
- later independent actions are still attempted;
- earlier successful actions are not rolled back;
- any failed action makes the command return nonzero;
- retry requires fresh observation and a new exact artifact.

A result such as `APPLIED, FAILED, APPLIED` is valid and intentionally non-atomic.

## Execution evidence

Artifact v2 execution writes execution-record v3:

```text
.repora/journal/<uid>--<execution-id>--intent.json
.repora/journal/<uid>--<execution-id>--result.json
```

One command execution ID is shared across selected repositories; each repository writes its own immutable pair. Each action preserves provider/path refs, before/desired/after OIDs, force intent, outcome, and sanitized error.

Intent failure prevents mutation. Result-write failure remains nonzero even when pushes completed. Journals are evidence, never replay authority.

## Public contracts

Current public envelopes include:

- `repora.status` v2;
- compatibility `repora.plan` v1 for the legacy view;
- exact reconciliation artifact v2, with artifact v1 historical support;
- `repora.apply` v2 for single-mirror-only command selections;
- `repora.apply` v3 for mixed or multi-mirror selections;
- execution-record v3, with v1/v2 historical parsing support.

Apply v3 exposes one ordered result per action with stable source/target, force, before, desired, optional after, outcome, and sanitized error.

## Concurrency and atomicity

Selected repositories use bounded concurrency after global preparation and force authorization. Each repository has its own intent/preflight/execution/result boundary. Mirrors inside a repository are sequential.

There is no cross-repository or cross-remote transaction. Repora performs no automatic rollback. A future concurrent mirror executor requires a separate decision because it would change ordering, evidence, and failure behavior.

## Release and assurance boundary

The mirror-controller and cross-platform packaging paths are complete in code. Security assurance includes reachable-vulnerability scanning, CodeQL, Git-history secret detection, dependency-license validation, workflow policy, race-enabled failure-path tests, and deterministic package verification.

The final v0.1 gate is operational rather than architectural:

- merge the release-hardening controls and documentation;
- complete the release checklist and curated changelog entry;
- publish the first protected `v0.1.0` tag from reviewed `main`;
- independently download and verify the published assets.

A repository-wide benchmark gate is explicitly deferred until a stable workload and useful threshold exist. Anthesis policy integration, managed artifacts, advanced document routing, and assessments remain outside the v0.1 path and must reuse the current plan, policy, execution, result, and evidence boundaries if later pursued.
