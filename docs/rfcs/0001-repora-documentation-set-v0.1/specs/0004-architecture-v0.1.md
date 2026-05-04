# SPEC-0004: Architecture (v0.1)

Status: Draft

## Execution Flow

```text
Observe -> Diff -> Plan -> Apply
```

## Components

### Spec Loader

Parses `repora.yaml`, validates required fields, normalizes defaults, and
produces an in-memory repository specification.

### Git Engine

Encapsulates shell execution of system Git commands. It is responsible for
clone, remote configuration, fetch, ref resolution, divergence comparison, and
mirror push operations.

### Observer

Materializes observed repository state from canonical and mirror remotes.
Observation should be side-effect-minimal apart from local cache updates and
remote fetches.

### Planner

Transforms desired state and observed state into a deterministic action plan.
The planner shall remain pure and side-effect-free.

### Executor

Applies approved plans. The executor owns side effects, including mirror pushes
and future destructive actions gated by `--force`.

## Concurrency Shape

Each repository should be executable as an isolated unit of work. Future
concurrent execution should use a bounded worker pool and deterministic result
ordering.
