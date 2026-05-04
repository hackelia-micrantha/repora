# ADR-0006: Storage Model

Status: Draft

## Decision

Repora v0.1 shall use local bare mirrors as its working representation for
repository state.

## Default Path

```text
~/.cache/repora/<repo>.git
```

## Disk Strategy (v0.1)

- Use full-history bare mirrors to preserve correctness for divergence
  detection
- Avoid shallow clones and filtered clones in v0.1
- Fetch with pruning so deleted upstream refs do not remain indefinitely in
  local cache state
- Run garbage collection only where it is safe for the selected storage mode

## Future Working Directory Management

A future mode may manage non-bare working directories for local developer
workflows. That mode must explicitly address:

- Dirty working tree state
- Checkout isolation
- Worktree lifecycle
- Conflict between Repora-managed state and human-edited local state
