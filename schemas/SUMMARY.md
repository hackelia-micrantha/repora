# Schema summary

Status: Current

## Purpose

This subtree contains versioned machine-readable compatibility contracts for Repora CLI output, reconciliation plans, execution records, document routing, context receipts, repository assessment artifacts, managed repository artifact planning/execution evidence, normalized repository posture inventory, deterministic documentation posture observations, mirror posture observations, hooks/local-workflow posture observations, and bounded commit-history posture observations.

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
- `posture-inventory-v1.schema.json`
- `posture-documentation-v1.schema.json`
- `posture-documentation-profile-v1.schema.json`
- `posture-mirrors-v1.schema.json`
- `posture-hooks-v1.schema.json`
- `posture-hooks-profile-v1.schema.json`
- `posture-commits-v1.schema.json`
- `posture-commits-profile-v1.schema.json`
- `document-router.schema.json`
- `context-receipt-v1.schema.json`
- `repository-assessment-v1.schema.json`
- `repository-snapshot-v1.schema.json`
- `finding-v1.schema.json`
- `evidence-v1.schema.json`
- `scorecard-v1.schema.json`

Migration guidance for public CLI contracts lives under [`../docs/cli/`](../docs/cli/). Repository/CI posture fact-state and collection semantics live in [`../docs/posture-inventory.md`](../docs/posture-inventory.md); documentation posture/profile semantics live in [`../docs/posture-documentation.md`](../docs/posture-documentation.md); mirror posture semantics live in [`../docs/posture-mirrors.md`](../docs/posture-mirrors.md); hooks/local-workflow and profile semantics live in [`../docs/posture-hooks.md`](../docs/posture-hooks.md); bounded commit-history and profile semantics live in [`../docs/posture-commits.md`](../docs/posture-commits.md). Assessment semantics and evidence-strength guidance live in [`../docs/assessments.md`](../docs/assessments.md). Managed artifact semantics live in [`../docs/architecture/managed-artifacts.md`](../docs/architecture/managed-artifacts.md) and ADR-0017.

## Ownership boundaries

Schemas define serialized structure and compatibility. They do not define the full operational semantics of planning, execution, routing, recovery, posture interpretation, assessment interpretation, or managed artifact mutation; use the corresponding architecture, posture, routing, assessment, or decision documents for those behaviors.

## Expand when

Load the exact schema whenever validating fields, required properties, enums, compatibility versions, or consumer migrations. Pair it with the owning architecture, posture, routing, assessment, or decision document when interpreting semantics.

## Exclusions and stale areas

Do not assume a higher version is interchangeable with an older consumer. Historical schemas remain valid evidence for their own version but are not substitutes for the requested contract version. Posture inventories are observed evidence, not findings or risk scores. Documentation observation profiles choose deterministic facts to collect; they do not define severity, suppress policy, or grant remediation authority. Mirror posture v1 remains default-branch scoped: tag and release drift are explicit unknown facts rather than inferred results. Hooks posture profiles select bounded observation expectations only: local hooks are early feedback, CI remains authoritative, and no hook code is installed or executed. Commit posture profiles select bounded history, sensitive paths, and deterministic thresholds only; they do not define severity, infer developer intent, or authorize productivity/identity analytics. Assessment scorecards are scoped evidence summaries, not objective whole-project grades unless the report explicitly supports that scope. Managed artifact plan, execution-record, and apply-result schemas define review/evidence contracts; mutation safety is governed by the managed-artifact architecture and exact-plan preflight/lease rules.
