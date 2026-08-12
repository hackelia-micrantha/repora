# Repository assessments and evidence

Status: Current

## Purpose

Repora assessment artifacts capture a point-in-time engineering review without replacing GitHub as the source of truth for code, issues, pull requests, commits, or work status.

The assessment framework provides versioned contracts, templates, a concrete example, local validation, bounded skeleton creation, and helper commands for findings and scorecards. It does not create or update GitHub issues or pull requests, modify repository source, enforce CI policy, or invoke Anthesis governance.

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

## Assessment lifecycle

1. Create a new skeleton at an explicit path:

   ```sh
   repoctl assess path/to/assessment.json
   ```

   `assess` creates the canonical v1 repository-assessment template. It does not inspect the current repository, load `repora.yaml`, call Git, or use the network. The target must not already exist; existing files and symlinks are never overwritten.

2. Replace the placeholders with the repository revision, scope, findings, evidence, and evidence-backed score rationale being reviewed.
3. Validate the complete artifact:

   ```sh
   repoctl validate-report path/to/assessment.json
   ```

4. Inspect findings and scorecard projections as needed with `list-findings` and `generate-scorecard`.
5. When repository freshness matters, create or update an assessment for the new reviewed revision. Older assessments remain point-in-time evidence and do not silently inherit newer GitHub state.

The generated skeleton is semantically locked by tests to `templates/assessments/repository-assessment-v1.json` so CLI and checked-in template contracts cannot drift independently.

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

List findings from a validated report with:

```sh
repoctl list-findings path/to/assessment.json
```

`list-findings` validates the complete v1 report before producing output. Findings remain in report order; the command does not re-rank or infer priority. Each output row is tab-separated as:

```text
ID<TAB>SEVERITY<TAB>STATUS<TAB>TYPE<TAB>JSON-QUOTED-TITLE
```

Quoting the title keeps embedded tabs, newlines, quotes, or other escaped characters from changing row boundaries. A valid report with no findings exits successfully with no output. A separate stabilized JSON listing contract is intentionally deferred rather than implicitly exposing the in-memory Go representation.

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

Render the scorecard already present in a validated report with:

```sh
repoctl generate-scorecard path/to/assessment.json
```

Despite the command name, `generate-scorecard` does not calculate, average, infer, normalize, or re-rank scores. It projects the validated scorecard in report order. Each output row is tab-separated as:

```text
DIMENSION<TAB>SCORE<TAB>JSON-EVIDENCE-ID-ARRAY<TAB>JSON-QUOTED-RATIONALE
```

This keeps the report itself authoritative for score values and evidence linkage. A score of `0` is preserved as a valid explicit value rather than treated as missing. Machine-readable command JSON remains deferred until a dedicated compatibility contract is justified.

## Templates and examples

- `templates/assessments/repository-assessment-v1.json` is the canonical general assessment skeleton and is generated by `repoctl assess`.
- `templates/assessments/qart-review-v1.json` provides Question, Answer/Finding, Recommendation, and Tradeoff placeholders.
- `examples/repository-assessment-v1.json` is a point-in-time routing review with explicit QART question/finding/recommendation/tradeoff items, a risk, a gap, classified evidence strength, traceable references, and scoped scorecard rationale. Tests lock those example concepts as part of the assessment acceptance contract.

Architecture-review, security-review, and resume-evidence-specific templates are explicitly deferred. The general assessment and QART contracts already represent those scopes without creating parallel artifact types before concrete use cases justify them.

Template placeholder values are not findings or evidence and must be replaced before a report is treated as an assessment.

## Validation

Validate one report directly with the standalone CLI command:

```sh
repoctl validate-report path/to/assessment.json
```

`validate-report` is read-only and does not load `repora.yaml`, inspect remotes, or perform Git operations. It currently accepts only `repora.repository-assessment` version 1. Unknown JSON fields, unsupported versions, malformed timestamps, missing required fields, invalid vocabularies, duplicate IDs, and unresolved evidence links fail validation.

Run the repository validation set with:

```sh
make assessment-test
```

That target exercises the same Go parser and validator used by the production command against the committed example and templates. Normal Go tests additionally lock the generated skeleton to the committed template. The normal schema test verifies every committed `*.schema.json` document is valid JSON.

## Deferred work

The framework intentionally defers:

- specialized architecture/security/resume-evidence templates until a concrete workflow requires distinct structure;
- stabilized machine-readable projections for assessment helper commands until compatibility consumers exist;
- automatic repository mutation, PR creation, or CI policy enforcement;
- Anthesis governance integration.
