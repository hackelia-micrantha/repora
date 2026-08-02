# Repora documentation

This directory contains implementation guidance, architecture records, machine-readable contracts, and historical design material for Repora.

## Authority map

Use the smallest authoritative source that answers the question.

| Question | Authoritative source |
| --- | --- |
| What does Repora support today? | [`../README.md`](../README.md) |
| How is the current implementation structured? | [`architecture/current-system.md`](architecture/current-system.md) |
| What are the current failure, exit, and recovery semantics? | [`architecture/failure-semantics.md`](architecture/failure-semantics.md) |
| What work is active and what is deferred? | [`plans/current.md`](plans/current.md) and GitHub issues |
| Why was an architectural choice made? | [`decisions/README.md`](decisions/README.md) and the linked ADR |
| What JSON shape is valid? | The versioned files under [`../schemas`](../schemas) |
| How does CI run locally and on GitHub? | [`ci.md`](ci.md) |
| How is repository topology configured? | [`configuration/provider-path-topology-v1.md`](configuration/provider-path-topology-v1.md) |
| What are the mirror-controller semantics? | [`architecture/mirror-workflow-semantics.md`](architecture/mirror-workflow-semantics.md) |
| How does deterministic document routing work? | [`routing/document-routing.md`](routing/document-routing.md) |

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
- **Architecture documents** explain the current implementation and package ownership.
- **Schemas** define machine-readable compatibility contracts.
- **README files** provide orientation and current capability summaries, not detailed requirements.

## Maintenance rules

Update documentation in the same change when any of these change:

- package ownership or dependency direction;
- public CLI behavior or exit codes;
- JSON contract shape or version;
- mutation, stale-plan, partial-failure, or recovery semantics;
- supported topology, providers, transports, refs, or mirrors;
- an accepted architectural decision;
- the active implementation order or a release gate.

Do not mark a design complete because a model or schema exists. Completion requires the observable path, tests, and documentation required by the owning issue.

## Pull-request documentation check

Every pull request should answer these questions in its description or checklist:

- Did current behavior change?
- Did a public JSON or schema contract change?
- Did an architecture boundary change?
- Did failure or recovery behavior change?
- Is an ADR required or superseded?
- Does the active plan need updating?

A concise “not applicable” is sufficient when the change does not affect documentation.
