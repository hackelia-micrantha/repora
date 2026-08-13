# Schema summary

Status: Current

## Purpose

This subtree contains versioned machine-readable compatibility contracts for Repora CLI output, reconciliation plans, execution records, document routing, context receipts, repository assessment artifacts, and managed repository artifact planning/execution evidence.

## Canonical sources

Use the schema matching the exact artifact kind and version being produced or consumed. Current families include:

- `cli-status-v*.schema.json`
- `cli-plan-v*.schema.json`
- `cli-apply-v*.schema.json`
- `reconciliation-plan-v*.schema.json`
- `managed-artifact-plan-v1.schema.json`
- `managed-artifact-execution-record-v1.schema.json`
- `managed-artifact-apply-result-v1.schema.json`
- `execution-record-v*.schema.json`
- `document-router.schema.json`
- `context-receipt-v1.schema.json`
- `repository-assessment-v1.schema.json`
- `repository-snapshot-v1.schema.json`
- `finding-v1.schema.json`
- `evidence-v1.schema.json`
- `scorecard-v1.schema.json`

Migration guidance for public CLI contracts lives under [`../docs/cli/`](../docs/cli/). Assessment semantics and evidence-strength guidance live in [`../docs/assessments.md`](../docs/assessments.md). Managed artifact semantics live in [`../docs/architecture/managed-artifacts.md`](../docs/architecture/managed-artifacts.md) and ADR-0017.

## Ownership boundaries

Schemas define serialized structure and compatibility. They do not define the full operational semantics of planning, execution, routing, recovery, assessment interpretation, or managed artifact mutation; use the corresponding architecture, routing, assessment, or decision documents for those behaviors.

## Expand when

Load the exact schema whenever validating fields, required properties, enums, compatibility versions, or consumer migrations. Pair it with the owning architecture, routing, assessment, or decision document when interpreting semantics.

## Exclusions and stale areas

Do not assume a higher version is interchangeable with an older consumer. Historical schemas remain valid evidence for their own version but are not substitutes for the requested contract version. Assessment scorecards are scoped evidence summaries, not objective whole-project grades unless the report explicitly supports that scope. Managed artifact plan, execution-record, and apply-result schemas define review/evidence contracts; mutation safety is governed by the managed-artifact architecture and exact-plan preflight/lease rules.
