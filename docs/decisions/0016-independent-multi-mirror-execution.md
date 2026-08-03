# ADR-0016: execute multi-mirror actions independently in exact plan order

Status: Implemented

Decision date: 2026-08-02

Review date: 2026-08-02

Implementation: PR #80

Related issues: #15

## Context

Repora can observe, plan, bind, and preflight several mirror actions before action zero. Real mutation must now represent an unavoidable property of independent Git remotes: one push may fail after an earlier push succeeded.

Pretending that several remotes form a transaction would hide partial success, encourage unsafe replay, and require rollback behavior that Git providers cannot guarantee consistently.

Apply output v2 also lacks explicit per-target outcome and before/desired/after evidence.

## Decision

After complete repository-level preflight succeeds, Repora will:

1. execute actions sequentially in exact artifact order;
2. bind each provider/path target to its current runtime alias;
3. use a normal push for normal intent;
4. use force-with-lease only for an action already marked forced and globally authorized by `--force`;
5. mark each successful action `APPLIED` with its desired OID as `after`;
6. mark a runtime push failure `FAILED` and continue to later independent actions;
7. never roll back an earlier successful mirror;
8. return nonzero if any action or journal operation fails;
9. persist one repository-level execution-record v3 result containing all outcomes;
10. require fresh observation and planning for retry.

Any selected configuration containing a multi-mirror repository uses `repora.apply` version 3. Single-mirror-only commands retain apply v2 compatibility.

Before any selected repository mutates, all selected repositories must complete observation and exact artifact preparation, and command-level force authorization must succeed. Complete topology, branch, policy, and OID preflight remains a repository execution boundary rather than a cross-repository transaction.

## Alternatives considered

### Stop after the first runtime failure

Rejected. Later mirrors are independent and may still be safely updated. Stopping would turn one provider outage into avoidable drift elsewhere.

### Roll back successful earlier mirrors

Rejected. A reverse push may itself fail, race with new work, require destructive authorization, or overwrite valid state. Rollback would create a second unreviewed plan.

### Execute mirrors concurrently

Rejected for the first implementation. Sequential execution preserves deterministic order and simplifies evidence, debugging, and recovery.

### Keep apply output v2

Rejected. Partial multi-target execution requires explicit outcomes and before/desired/after evidence. Altering v2 in place would break its committed contract.

### Make all selected repositories atomic

Rejected. Repository-level preparation before mutation reduces preventable partial work, but independent repositories and remotes do not form a transaction.

## Consequences

- a single repository result can honestly contain `APPLIED`, `FAILED`, `APPLIED`;
- successful mirrors remain updated after another mirror fails;
- operators can identify exactly which targets need fresh reconciliation;
- command status remains nonzero for partial success;
- journal result failure remains nonzero even when pushes completed;
- apply v2 and v3 coexist by documented topology boundary;
- retry is always a new plan, never replay or rollback.

## Security implications

Complete stale-ref preflight still occurs before action zero. Forced actions retain their reviewed lease OID and cannot be introduced by the command flag. Runtime error diagnostics are sanitized before public output and journal serialization.

Continuing after failure does not widen authority: each later action was already present in the exact reviewed artifact, matched to current provider/path topology, and preflighted.

## Compatibility

- reconciliation artifact v2 remains the exact execution input;
- execution-record v3 remains the path-bound evidence format;
- apply v3 is additive and does not change apply v2;
- single-mirror-only workflows retain their existing public result contract;
- mixed or multi-mirror selections use apply v3 consistently;
- no journal or result is replay authority.

## Validation

Implementation must demonstrate:

- all target refs are preflighted before the first push;
- the middle of three mirrors can fail while the first and third apply;
- forced actions use the current runtime alias and reviewed force-with-lease OID;
- preparation and missing-force failures perform zero mutation and write no intent;
- apply v3 and execution-record v3 preserve ordered per-target outcomes;
- multiple local bare remotes reproduce real partial success;
- no documentation implies rollback or cross-remote atomicity;
- formatting, vet, race, integration, cross-platform, vulnerability, and CodeQL checks pass.
