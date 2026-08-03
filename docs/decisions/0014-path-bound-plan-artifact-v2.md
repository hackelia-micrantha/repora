# ADR-0014: Provider-path-bound reconciliation artifact v2

Status: Implemented

Decision date: 2026-08-02

Last reviewed: 2026-08-02

Supersedes: none

Superseded by: none

Implemented by: PR #77

Related issues: #8, #13, #15

## Context

Reconciliation artifact version 1 identifies refs by provider, runtime Git alias, and branch. That is adequate for one configured mirror but cannot bind an action to a durable mirror target when several mirrors exist or configuration order changes.

Using array index or aliases such as `mirror-0` as identity would allow reordering to retarget a reviewed artifact. Using resolved URLs would mix transport with identity and could expose irrelevant or sensitive details.

## Decision

New production plans emit reconciliation artifact version 2.

Every source and target ref includes a provider-relative repository path in addition to provider, runtime alias, and branch. Provider/path is the durable topology identity. Runtime aliases remain execution details.

Version 2 validates paths strictly and rejects missing, absolute, traversal-bearing, whitespace-bearing, transport-like, and credential-like values.

Version 1 remains parseable for historical single-mirror compatibility. It is never interpreted as path-bound or valid for future multi-mirror targeting.

The CLI remains single-mirror for mutation in this slice. The gate is removed only after multi-target planning, complete preflight, independent outcomes, and journal evidence are implemented.

## Alternatives

### Use mirror array index

Rejected because ordering is not identity and configuration reordering would retarget reviewed actions.

### Use runtime Git aliases

Rejected because aliases are locally assigned execution state and may change when topology changes.

### Use resolved URLs

Rejected because transport is mutable, may contain sensitive data, and is not durable logical identity.

### Break version 1 parsing

Rejected because existing exported single-mirror plans can remain safely supported under their historical provider/alias boundary.

### Enable multi-mirror apply in the same slice

Rejected because artifact identity should stabilize independently before executor continuation, result, and journal contracts change.

## Consequences

- New exact plans have stable provider/path target binding.
- Imported v2 artifacts fail before Git reads when topology paths do not match configuration.
- Historical v1 artifacts remain usable only for the existing single-mirror path.
- Execution-record v2 can reference either plan artifact version through its version and digest.
- Multi-mirror apply still requires a follow-up contract migration.

## Security implications

- Reviewed targets cannot be redirected by mirror reordering.
- URLs, credentials, local paths, and transport aliases are not durable identity.
- Absolute and traversal paths fail artifact validation.
- The existing default-branch policy, force authorization, stale preflight, and journal requirements remain unchanged.

## Validation

- v2 golden contract and schema;
- v1 parsing and fixture compatibility;
- provider-path round-trip tests;
- unsafe and absolute path rejection tests;
- topology mismatch rejection before Git reads;
- local transport integration with separate declarative identity;
- journal references to v1 and v2 artifacts.
