# REQ-0001: Core System Requirements

Status: Draft

## Functional Requirements

Repora v0.1 shall:

- Represent each managed repository through a stable declarative identity in
  `repora.yaml`
- Designate exactly one canonical remote per repository
- Represent one or more mirror remotes as replication targets
- Mirror canonical repository state to target repositories using full-ref mirror
  semantics
- Detect observable repository-state relationships between canonical and mirror
  remotes, including equal, canonical-ahead, mirror-ahead, and divergent
  histories
- Expose repository observation, planning, and execution workflows through
  `repoctl`
- Support an explicit destructive override for mirror reconciliation when the
  operator intentionally chooses canonical state over mirror-side drift

## Non-Functional Requirements

Repora v0.1 shall be designed around the following operational properties:

- **Idempotence**: repeated execution against unchanged repository state should
  produce no additional side effects
- **Determinism**: equivalent inputs and observed remote state should yield
  equivalent plans
- **External truth**: Git remotes and the local bare mirror are the authoritative
  state surfaces; Repora shall not introduce an internal database in v0.1
- **CLI-first operation**: human and automation usage shall both be supported
  from the command line
- **Concurrency-safe architecture**: implementation shall avoid global mutable
  state and preserve per-repository isolation, even if v0.1 execution remains
  sequential

## Safety Requirements

- Repora shall fail closed on divergence by default
- `--force` shall overwrite mirror refs from canonical state; it shall never
  mutate canonical refs
- Destructive behavior shall require explicit operator intent
- Mirror-side changes shall be treated as drift unless explicitly accepted
  through a future policy mechanism
