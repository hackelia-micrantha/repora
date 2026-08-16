# Repora documentation

This directory contains implementation guidance, architecture records, machine-readable contracts, and historical design material for Repora.

## Authority map

Use the smallest authoritative source that answers the question.

| Question | Authoritative source |
| --- | --- |
| What does Repora support today? | [`../README.md`](../README.md) |
| What changed for users and operators? | [`../CHANGELOG.md`](../CHANGELOG.md) |
| How do I install and verify a release? | [`release.md`](release.md) |
| What is required to publish and verify a release? | [`release-checklist.md`](release-checklist.md) |
| How do I build/run/compose Repora with Nix? | [`nix.md`](nix.md) |
| What security checks run and how are findings handled? | [`security-ci.md`](security-ci.md) |
| How do repository assessments and evidence work? | [`assessments.md`](assessments.md) |
| How does the GitHub repository/CI posture inventory work? | [`posture-inventory.md`](posture-inventory.md) |
| How does deterministic documentation/README posture work? | [`posture-documentation.md`](posture-documentation.md) |
| What is the broader repository/CI posture model? | [`posture.md`](posture.md) |
| Why is there no repository-wide benchmark gate? | [`benchmarks.md`](benchmarks.md) |
| How is the current implementation structured? | [`architecture/current-system.md`](architecture/current-system.md) |
| What is the managed-artifact/README mutation boundary? | [`architecture/managed-artifacts.md`](architecture/managed-artifacts.md) and [ADR-0017](decisions/0017-managed-artifact-domain.md) |
| How does managed README planning and apply work? | [`architecture/managed-artifact-planning.md`](architecture/managed-artifact-planning.md) |
| What serialized contract defines managed README review plans? | [`architecture/managed-artifact-plan-v1.md`](architecture/managed-artifact-plan-v1.md) |
| How does multi-mirror status work? | [`architecture/multi-mirror-status.md`](architecture/multi-mirror-status.md) |
| How does the exact mirror plan artifact work? | [`architecture/reconciliation-plan-artifact.md`](architecture/reconciliation-plan-artifact.md) |
| How do plan-artifact consumers migrate to v2? | [`cli/plan-artifact-v2.md`](cli/plan-artifact-v2.md) |
| How does execution evidence work? | [`architecture/execution-journal.md`](architecture/execution-journal.md) |
| How do execution-record consumers migrate to v3? | [`cli/execution-record-v3.md`](cli/execution-record-v3.md) |
| How do apply consumers migrate to per-target v3 results? | [`cli/apply-v3.md`](cli/apply-v3.md) |
| What are the current failure, exit, and recovery semantics? | [`architecture/failure-semantics.md`](architecture/failure-semantics.md) |
| What work is active and what is deferred? | [`plans/current.md`](plans/current.md) and GitHub issues |
| Why was an architectural choice made? | [`decisions/README.md`](decisions/README.md) and the linked ADR |
| What JSON shape is valid? | The versioned files under [`../schemas`](../schemas) |
| How do status consumers migrate to v2? | [`cli/status-v2.md`](cli/status-v2.md) |
| How does CI run locally and on GitHub? | [`ci.md`](ci.md) |
| How is repository topology configured? | [`configuration/provider-path-topology-v1.md`](configuration/provider-path-topology-v1.md) |
| What are the mirror-controller semantics? | [`architecture/mirror-workflow-semantics.md`](architecture/mirror-workflow-semantics.md) |
| How does deterministic document routing work? | [`routing/document-routing.md`](routing/document-routing.md) |
| What optional Anthesis policy boundary is accepted? | [`architecture/anthesis-policy-integration.md`](architecture/anthesis-policy-integration.md) and [ADR-0018](decisions/0018-optional-anthesis-policy-gate.md) |

Source code and tests remain the final authority when documentation and implementation disagree. A disagreement is a documentation defect and should be fixed rather than treated as an alternative interpretation.

## Document classes

### Current

Current documents describe merged behavior, accepted decisions, active plans, or published contracts. They must remain aligned with source and tests.

### Proposed

Proposed documents describe a design that has not been accepted or implemented. They must state `Status: Proposed` and must not be written as current behavior.

### Historical

Historical documents preserve earlier reasoning but are not implementation authority. They should state what superseded them and link to the current source.

The RFC-0001 documentation set and `plans/implementation-plan-v0.1.md` predate several architecture and CI changes. They are retained for history. Use the authority map above for current behavior.

## Source-of-truth boundaries

- **GitHub issues** own work state, acceptance criteria, and implementation tracking.
- **Plans** own ordering, release gates, and explicit deferrals; they do not duplicate every issue subtask.
- **ADRs** own durable decisions and consequences; they do not act as project plans.
- **Architecture documents** explain current implementation, authority boundaries, and package ownership.
- **Schemas** define machine-readable compatibility contracts.
- **Posture inventories** capture normalized observed/unknown/unavailable repository evidence; repository-owned observation profiles select deterministic facts but do not define severity, findings, or remediation policy.
- **Assessment reports** capture point-in-time analysis and evidence; they reference GitHub/repository state rather than replacing it.
- **README files** provide orientation and current capability summaries, not detailed requirements.
- **The changelog** records curated user-visible and compatibility changes; generated release notes provide commit and contributor detail.
- **Release checklists** define accountable publication evidence but do not replace CI or implementation tests.

## Maintenance rules

Update documentation in the same change when any of these change:

- package ownership or dependency direction;
- public CLI behavior or exit codes;
- JSON contract shape or version;
- mutation, stale-plan, partial-failure, or recovery semantics;
- posture fact/evidence semantics, provider observation boundaries, or repository-owned observation profile contracts;
- supported topology, providers, transports, refs, mirrors, or managed artifact types;
- supported release targets, packaging, installation, or verification behavior;
- security checks, suppression rules, or release-blocking policy;
- an accepted architectural decision;
- the active implementation order or a release gate.

Do not mark a design complete because a model or schema exists. Completion requires the observable path, tests, and documentation required by the owning issue.

## Pull-request documentation check

Every pull request should answer these questions in its description or checklist:

- Did current behavior change?
- Did a public JSON or schema contract change?
- Did an architecture/authority boundary change?
- Did failure or recovery behavior change?
- Did release packaging or supported targets change?
- Did security validation or release policy change?
- Is an ADR required or superseded?
- Does the active plan need updating?

A concise “not applicable” is sufficient when the change does not affect documentation.
