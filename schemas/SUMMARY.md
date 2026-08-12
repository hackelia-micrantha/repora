# Schema summary

Status: Current

## Purpose

This subtree contains versioned machine-readable compatibility contracts for Repora CLI output, reconciliation plans, execution records, document routing, context receipts, and repository assessment artifacts.

## Canonical sources

Use the schema matching the exact artifact kind and version being produced or consumed. Current families include:

- `cli-status-v*.schema.json`
- `cli-plan-v*.schema.json`
- `cli-apply-v*.schema.json`
- `reconciliation-plan-v*.schema.json`
- `execution-record-v*.schema.json`
- `document-router.schema.json`
- `context-receipt-v1.schema.json`
- `repository-assessment-v1.schema.json`
- `repository-snapshot-v1.schema.json`
- `finding-v1.schema.json`
- `evidence-v1.schema.json`
- `scorecard-v1.schema.json`

Migration guidance for public CLI contracts lives under [`../docs/cli/`](../docs/cli/). Assessment semantics and evidence-strength guidance live in [`../docs/assessments.md`](../docs/assessments.md).

## Ownership boundaries

Schemas define serialized structure and compatibility. They do not define the full operational semantics of planning, execution, routing, recovery, or assessment interpretation; use the corresponding architecture, routing, or assessment documents for those behaviors.

## Expand when

Load the exact schema whenever validating fields, required properties, enums, compatibility versions, or consumer migrations. Pair it with the owning architecture, routing, or assessment document when interpreting semantics.

## Exclusions and stale areas

Do not assume a higher version is interchangeable with an older consumer. Historical schemas remain valid evidence for their own version but are not substitutes for the requested contract version. Assessment scorecards are scoped evidence summaries, not objective whole-project grades unless the report explicitly supports that scope.
