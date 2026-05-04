# ADR-0008: Disk Usage Optimization

Status: Draft

## Decision

Repora shall prioritize full-history correctness in v0.1 and optimize disk usage
through pruning, garbage collection, and optional shared object storage rather
than shallow history.

## Techniques

### Pruning and Garbage Collection

Repora should prune stale refs during fetch and may reclaim unreachable objects
when safe:

```text
git fetch --prune <remote>
git gc --prune=now
```

### Shared Object Store (future / advanced)

A future storage mode may deduplicate objects across repositories through Git
alternates.

Proposed layout:

```text
~/.cache/repora/objects/
~/.cache/repora/repos/<repo>.git
```

Per-repository mirrors may reference shared objects through:

```text
.git/objects/info/alternates
```

## Alternates Safety Constraints

- Alternates must be treated as read-only object dependencies from the child
  repository perspective
- Repora must avoid garbage-collecting shared objects that are still reachable
  from any dependent repository
- Child repositories using alternates should not perform aggressive repacks that
  obscure dependency relationships
- Recovery procedures must document how to repack a repository into a
  self-contained state if alternates become unavailable

## Large File Storage

Repositories using Git LFS require explicit handling. Full Git ref mirroring
does not necessarily guarantee complete LFS object availability. Future support
should specify whether Repora runs:

```text
git lfs fetch --all
git lfs push --all <mirror>
```

## Consequences

- Full-history mirrors consume more disk than shallow clones
- Correctness remains stronger because divergence detection can inspect history
- Object deduplication can reduce disk usage but increases storage-management
  complexity
