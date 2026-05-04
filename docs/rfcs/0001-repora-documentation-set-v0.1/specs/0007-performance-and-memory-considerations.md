# SPEC-0007: Performance and Memory Considerations

Status: Draft

## Process Model

Repora v0.1 shells out to Git rather than embedding a Git implementation. This
reduces memory pressure inside Repora itself and delegates packfile handling,
object negotiation, and transport behavior to mature Git tooling.

## Memory Constraints

Repora should stream command output where practical and avoid loading large Git
output into memory unless bounded. JSON result construction should aggregate
summarized repository state, not raw command logs.

## Disk Constraints

Full-history mirrors are intentionally disk-heavy. Disk usage should be
mitigated through pruning, conservative garbage collection, and future shared
object storage rather than correctness-reducing shallow clones.

## Concurrency Constraints

Concurrent execution must be bounded. A future `--parallel N` implementation
should default conservatively and avoid unbounded simultaneous fetches, which can
saturate disk, network, file descriptors, and remote provider rate limits.
