# Architecture summary

Status: Current

## Purpose

This subtree describes Repora's implemented mirror-controller architecture: topology and transport, observation, ref policy, deterministic planning, apply orchestration, execution evidence, and failure semantics.

## Canonical sources

- [`current-system.md`](current-system.md) — package ownership and current implementation structure.
- [`mirror-workflow-semantics.md`](mirror-workflow-semantics.md) — mirror-controller operating semantics.
- [`reconciliation-plan-artifact.md`](reconciliation-plan-artifact.md) — exact executable plan contract.
- [`execution-journal.md`](execution-journal.md) — durable intent and result evidence.
- [`failure-semantics.md`](failure-semantics.md) — partial failure, stale state, exit, and recovery behavior.
- [`ref-policy.md`](ref-policy.md) — supported ref-policy boundary.
- [`multi-mirror-status.md`](multi-mirror-status.md) — independent mirror observation.

## Proposed architecture

- [`repository-topology-and-trust-domains.md`](repository-topology-and-trust-domains.md) — **Proposed** post-v0.2 topology model for hosted endpoints, trust domains, projection, promotion, contribution import, and archive semantics under #149/#150. It is not implemented behavior until the linked contracts land.

## Ownership boundaries

Architecture documents explain current design and package responsibilities. Versioned schemas define serialized compatibility contracts; source and tests define executable behavior.

## Expand when

Load the referenced document when exact ordering, safety, mutation, concurrency, compatibility, stale-plan, or recovery behavior matters. Load implementation packages as well when validating that documentation still matches code.

## Exclusions and stale areas

Do not infer atomic multi-remote transactions, rollback, arbitrary-ref support, hosted-control-plane behavior, projection, promotion, public contribution import, archive execution, or generic Forgejo support from this summary. Proposed documents remain non-executable until their implementation issues land.
