# Documentation summary

Status: Current

## Purpose

This subtree explains Repora's current architecture, CLI contracts, configuration, routing, release process, decisions, plans, and historical design material.

## Canonical sources

- [`README.md`](README.md) — documentation authority map and maintenance rules.
- [`../README.md`](../README.md) — current product capabilities and operator-facing behavior.
- [`architecture/current-system.md`](architecture/current-system.md) — current implementation structure.
- [`plans/current.md`](plans/current.md) — active work and explicit deferrals.
- [`../schemas/`](../schemas/) — machine-readable compatibility contracts.

## Ownership boundaries

Documentation explains behavior and decisions. Source code and tests remain final authority when documentation disagrees with implementation. GitHub issues own work state and acceptance criteria.

## Expand when

Load deeper documents when the question concerns exact CLI/schema compatibility, architecture boundaries, failure/recovery semantics, security controls, release behavior, configuration, routing, or an accepted decision.

## Exclusions and stale areas

Historical RFC material and superseded implementation plans are retained for traceability and are not current implementation authority. Use the authority map in [`README.md`](README.md) before relying on historical documents.
