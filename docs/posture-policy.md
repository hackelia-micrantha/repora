# Posture policy and deterministic reports

Status: Current

## Purpose

Repora's posture collectors produce normalized repository facts. The policy layer consumes those facts and evaluates explicit expectations without re-scanning repositories or calling provider APIs.

The v1 convergence path is intentionally decomposable:

```text
normalized fact -> policy expectation -> evaluation -> evidence/remediation -> report
```

It does not calculate an opaque repository score.

## Public contracts

Three versioned JSON contracts define the boundary:

- `repora.posture-policy-profile` v1 — external rules, severity, remediation, and exceptions;
- `repora.posture-policy-inputs` v1 — normalized facts using the shared `observed`, `unknown`, and `unavailable` evidence states;
- `repora.posture-report` v1 — deterministic evaluations and report metadata.

Schemas:

- [`../schemas/posture-policy-profile-v1.schema.json`](../schemas/posture-policy-profile-v1.schema.json)
- [`../schemas/posture-policy-inputs-v1.schema.json`](../schemas/posture-policy-inputs-v1.schema.json)
- [`../schemas/posture-report-v1.schema.json`](../schemas/posture-report-v1.schema.json)

## CLI

Policy evaluation is offline-only:

```text
repoctl posture report \
  --profile policy.json \
  --facts posture-facts.json \
  --as-of 2026-08-23 \
  --format markdown
```

Use `--format json` for the versioned `repora.posture-report` v1 artifact.

`--as-of` is required and must use `YYYY-MM-DD`. Exception expiry therefore has no hidden wall-clock dependency: the same profile, facts, and evaluation date produce the same result.

The command does not use `GITHUB_TOKEN`, contact providers, fetch repository content, run scanners, or mutate repository/provider state.

## Policy profile

A policy profile is external policy data. Repora does not automatically load it from the target repository, and repository-owned observation profiles do not become policy authority.

Example:

```json
{
  "kind": "repora.posture-policy-profile",
  "version": 1,
  "id": "baseline",
  "rules": [
    {
      "id": "default-branch-protected",
      "area": "repository",
      "fact": "repository.default_branch_protected",
      "operator": "equals",
      "expected": true,
      "severity": "high",
      "title": "Default branch is protected",
      "remediation": ["Enable default-branch protection."]
    }
  ],
  "exceptions": []
}
```

Supported operators are:

- `equals` — exact JSON-value equality;
- `at_least` — numeric observed value is greater than or equal to expected;
- `at_most` — numeric observed value is less than or equal to expected;
- `non_empty` — string, array, or object is non-empty and does not take an `expected` value.

Severity is explicit policy data: `critical`, `high`, `medium`, `low`, or `informational`. A mismatch on informational policy is reported as `warning`; other mismatches are `fail`.

## Exceptions

A rule exception requires all of:

- rule ID;
- reason;
- owner;
- expiry date.

For a mismatching rule:

- an exception is active through its expiry date and produces `excepted`;
- on the next date, the original `fail` or `warning` returns and the report records `exception expired`.

Exceptions do not convert unknown or unavailable evidence into a pass. Missing evidence remains visible.

## Normalized fact inputs

`repora.posture-policy-inputs` is the convergence artifact. Each fact contains:

- state: `observed`, `unknown`, or `unavailable`;
- a JSON value only when observed;
- source evidence.

The Go convergence adapters consume the existing typed posture artifacts directly:

- `AddInventory` — `repora.posture-inventory` v1;
- `AddDocumentation` — `repora.posture-documentation` v1;
- `AddHooks` — `repora.posture-hooks` v1;
- `AddCommits` — `repora.posture-commits` v1;
- `AddMirrors` — one UID selected from a validated `repora.posture-mirrors` v1 inventory.

Adapters preserve fact state and evidence. They reject duplicate fact names and preflight all additions before mutating the convergence input. GitHub-derived domains also reject repository-identity mixing. Mirror input must first pass the existing mirror-inventory validation contract.

Representative fact namespaces are:

- `repository.*` and `ci.*`;
- `documentation.*`;
- `hooks.*`;
- `commits.*`;
- `mirrors.*`.

Dynamic workflow, document, hook, commit, and mirror-target identifiers remain part of the fact name so evidence can be correlated without creating a second scanner or drift algorithm.

## Unknown and unavailable evidence

Policy never silently turns evidence gaps into pass or fail.

- `unknown` means the current fact contract cannot prove the value or the normalized fact was not supplied;
- `unavailable` means the fact is in scope but could not be observed under the current provider/access conditions.

Both states are preserved in JSON and collected again in the Markdown report's `Unknown or unavailable evidence` section.

## Report semantics

`repora.posture-report` v1 records:

- repository identity;
- profile ID;
- explicit `as_of` evaluation date;
- one deterministic evaluation per rule;
- rule area, severity, status, and title;
- expected and observed values where applicable;
- source evidence;
- remediation options;
- exception details and expired-exception state.

Markdown output sorts areas and rules deterministically. The findings summary counts active `fail` and `warning` evaluations by severity. Active exceptions are shown as `excepted`; unknown/unavailable evidence is shown separately rather than hidden from the summary.

## Security and authority boundary

Policy profiles and normalized facts are strict data contracts, not executable configuration.

The posture policy layer has no authority to:

- execute repository code or hook configuration;
- call provider APIs;
- mutate branch protection, mirrors, files, tags, or releases;
- perform automatic PR remediation;
- run external scanners;
- infer missing facts from unrelated evidence;
- turn repository observation profiles into severity or exception policy;
- calculate a hidden whole-repository score.

Any future provider remediation remains a separate reviewed mutation capability.

## Relationship to repository assessments

The existing `repora.repository-assessment` contract remains useful for broader point-in-time analysis and evidence-backed assessment workflows. It also requires a scorecard, so the posture policy implementation deliberately preserves a separate no-opaque-score shape.

Posture policy reuses the same architectural preference for explicit findings and evidence while preserving its own no-opaque-score and normalized-fact boundaries. A future projection into an assessment artifact can be added without making assessment scoring a prerequisite for posture evaluation.
