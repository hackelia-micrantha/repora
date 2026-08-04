# Benchmark scope

Status: Current

Repora does not use a performance benchmark as a v0.1 release gate.

## Decision

The v0.1 critical path is correctness and safety around Git observation, exact planning, stale-ref validation, mutation, and durable evidence. End-to-end command time is currently dominated by system Git, filesystem state, repository history, credentials, and remote network latency. A benchmark on shared CI hardware would produce a precise-looking number without a stable workload or useful release threshold.

The benchmark requirement under the v0.1 hardening plan is therefore explicitly deferred rather than implemented as a noisy gate.

## Evidence retained for v0.1

The release still validates bounded operational behavior through:

- explicit Git subprocess timeouts and cancellation tests;
- bounded repository-level concurrency;
- repeated race-enabled test execution;
- full integration tests against disposable repositories;
- deterministic repeated release packaging;
- CLI and packaged-binary smoke tests;
- workflow job timeouts.

These controls detect hangs, runaway subprocesses, nondeterminism, and gross regressions more directly than a synthetic wall-clock threshold at the current maturity.

## Trigger for a benchmark suite

Add reviewed benchmarks when at least one of these becomes true:

- a user-visible latency or throughput objective is defined;
- repositories at a representative scale expose a repeatable bottleneck;
- internal planning or serialization cost becomes material relative to Git operations;
- concurrency policy changes;
- a performance regression cannot be protected by a focused behavior test;
- release size or startup time becomes a supported constraint.

A future benchmark must define:

- the exact fixture shape and repository scale;
- which work is local versus network-dependent;
- hardware and toolchain assumptions;
- warm/cold cache behavior;
- informational versus blocking thresholds;
- variance handling and reproduction commands.

Until those conditions exist, performance changes should be assessed with focused profiling or issue-specific measurements rather than a repository-wide CI benchmark gate.
