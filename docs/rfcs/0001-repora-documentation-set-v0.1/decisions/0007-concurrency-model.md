# ADR-0007: Concurrency Model

Status: Draft

## Decision

Repora shall be designed for concurrent per-repository execution, even if v0.1
initially executes sequentially.

## Requirements

- No shared mutable global repository state
- Per-repository cache isolation
- Bounded worker-pool design for future parallel execution
- Deterministic aggregation of results for stable output

## Future Flag

```text
--parallel N
```
