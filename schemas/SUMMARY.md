# Schema summary

Status: Current

## Purpose

This subtree contains versioned machine-readable compatibility contracts for Repora CLI output, reconciliation plans, execution records, document routing, and context receipts.

## Canonical sources

Use the schema matching the exact artifact kind and version being produced or consumed. Current families include:

- `cli-status-v*.schema.json`
- `cli-plan-v*.schema.json`
- `cli-apply-v*.schema.json`
- `reconciliation-plan-v*.schema.json`
- `execution-record-v*.schema.json`
- `document-router.schema.json`
- `context-receipt-v1.schema.json`

Migration guidance for public CLI contracts lives under [`../docs/cli/`](../docs/cli/).

## Ownership boundaries

Schemas define serialized structure and compatibility. They do not define the full operational semantics of planning, execution, routing, or recovery; use the corresponding architecture and routing documents for those behaviors.

## Expand when

Load the exact schema whenever validating fields, required properties, enums, compatibility versions, or consumer migrations. Pair it with the owning architecture or routing document when interpreting semantics.

## Exclusions and stale areas

Do not assume a higher version is interchangeable with an older consumer. Historical schemas remain valid evidence for their own version but are not substitutes for the requested contract version.
