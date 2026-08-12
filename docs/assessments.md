# Repository assessments and evidence

Status: Current

## Purpose

Repora assessment artifacts capture a point-in-time engineering review without replacing GitHub as the source of truth for code, issues, pull requests, commits, or work status.

The first assessment slice is intentionally data-only. It defines versioned contracts, templates, a concrete example, and local validation. It does not yet add assessment-generation CLI commands, scorecard automation, repository mutation, pull-request creation, or CI enforcement.

## Artifact model

A `repora.repository-assessment` contains:

- a repository snapshot identifying the exact revision reviewed;
- findings for questions, risks, gaps, overlap, drift, recommendations, and tradeoffs;
- evidence objects supporting explicit engineering claims;
- a scorecard whose dimensions cite evidence IDs;
- optional report metadata.

The component contracts are versioned independently:

- `repository-assessment-v1.schema.json`
- `repository-snapshot-v1.schema.json`
- `finding-v1.schema.json`
- `evidence-v1.schema.json`
- `scorecard-v1.schema.json`

## Source-of-truth boundary

Assessment reports reference GitHub state; they do not duplicate it as authoritative mutable state.

Use references for issues, pull requests, commits, files, or URLs. A report may explain why a reference matters, but issue status, PR status, code contents, and commit history remain owned by GitHub and the repository itself.

Every assessment must include the reviewed repository revision. A later repository state does not retroactively update an older assessment. Re-run or replace the assessment when freshness matters.

## Findings

Supported finding types are:

- `question`
- `finding`
- `recommendation`
- `tradeoff`
- `risk`
- `gap`
- `overlap`
- `drift`

Severity is one of `critical`, `high`, `medium`, `low`, or `informational`.

Finding status is one of `open`, `accepted`, `deferred`, `implemented`, or `rejected`. This status describes the finding inside the assessment artifact; it must not be used as a substitute for the live state of a linked GitHub issue or pull request.

## Evidence strength

Evidence categories describe the engineering area being supported. Strength describes how defensible the claim is from the cited repository material:

| Strength | Meaning |
| --- | --- |
| `strong` | Direct repository evidence clearly supports the claim. |
| `moderate` | Evidence supports the claim but requires limited interpretation or is incomplete. |
| `weak` | Evidence is indirect, narrow, or materially incomplete. |
| `unsupported` | The claim is retained for review but currently lacks supporting evidence. |

`strong`, `moderate`, and `weak` evidence must include at least one reference. `unsupported` evidence may have none. Evidence strength must not be inflated for resume or job-search use; the rationale should explain why the selected strength is defensible.

## Scorecards

Scorecards use integer values from 0 through 5 for bounded dimensions such as architecture, security, testing, delivery, operations, maintainability, and documentation.

Scores are not objective project grades. Each score requires a rationale and may cite evidence IDs. Scope must be explicit: a score supported only by routing evidence must not be presented as a whole-repository architecture score.

## Templates and examples

- `templates/assessments/repository-assessment-v1.json` is a valid skeleton for general reviews.
- `templates/assessments/qart-review-v1.json` provides Question, Answer/Finding, Recommendation, and Tradeoff placeholders.
- `examples/repository-assessment-v1.json` demonstrates traceable findings, evidence strength, and scoped scorecard rationale.

Template placeholder values are not findings or evidence and must be replaced before a report is treated as an assessment.

## Validation

Run:

```sh
make assessment-test
```

The dependency-free validator checks the committed example and templates for:

- artifact kind/version boundaries;
- snapshot revision shape;
- valid finding, severity, status, evidence, and score vocabularies;
- unique finding and evidence IDs;
- evidence references required by non-unsupported strength;
- finding evidence IDs resolving to declared evidence;
- unique scorecard dimensions;
- scorecard evidence IDs resolving to declared evidence.

The normal schema test also verifies every committed `*.schema.json` document is valid JSON.

## Deferred work

Issue #24 continues to own the broader assessment framework. This slice explicitly defers:

- `repora assess` report skeleton generation;
- `repora validate-report` CLI behavior;
- `repora list-findings`;
- `repora generate-scorecard`;
- automated architecture/security/resume-evidence templates beyond the repository and QART skeletons;
- automatic repository mutation, PR creation, or CI policy enforcement.
