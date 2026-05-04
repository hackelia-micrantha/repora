# ADR-0003: Divergence Handling

Status: Draft

## Decision

Repora shall fail by default when canonical and mirror histories diverge. The
operator may use `--force` to overwrite mirror refs from canonical state.

## Rationale

Divergence is a potentially destructive state. Automatic repair would risk
deleting legitimate mirror-side work or masking unauthorized changes. Failing by
default preserves forensic visibility and requires explicit human intent for
destructive reconciliation.

## Consequences

- Safe operation is the default behavior
- `--force` becomes an explicit trust and authority assertion by the operator
- Tests must validate that divergence exits with code `2` unless `--force` is
  present
