# Routing summary

Status: Current

## Purpose

This subtree defines deterministic document routing, subsystem manifest composition, trust tiers, route fixtures, context receipts, and hierarchical summary behavior.

## Canonical sources

- [`document-routing.md`](document-routing.md) — routing model and selection boundary.
- [`manifests.md`](manifests.md) — explicit subsystem manifest composition.
- [`trust-tiers.md`](trust-tiers.md) — authority taxonomy and eligibility rules.
- [`route-tests.md`](route-tests.md) — deterministic route regression contract.
- [`context-receipts.md`](context-receipts.md) — routed-context evidence contract.
- [`summaries.md`](summaries.md) — summary-first progressive retrieval contract.
- [`router.manifest.yaml`](router.manifest.yaml) — routing subsystem manifest.

## Ownership boundaries

Routing decides which repository evidence is eligible and selected. It does not make selected content authoritative, execute repository mutations, infer trust automatically, or replace canonical documents with summaries.

## Expand when

Load the relevant canonical routing document when changing route semantics, trust behavior, manifest composition, receipt fields, summary expansion, budgets, or security-sensitive eligibility rules. Load fixtures and validators when verifying behavior.

## Exclusions and stale areas

Generated, experimental, archived, and external material remains excluded unless explicitly requested by the routing policy. Summaries are derived orientation artifacts and must not be treated as canonical evidence.
