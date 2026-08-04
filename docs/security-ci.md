# Security CI and finding triage

Status: Current

Repora uses a small set of deterministic security checks appropriate to a local-first Go CLI that can mutate durable Git state. Security CI is a release gate, not a substitute for threat modeling or review of the mutation boundary.

## Checks

| Check | Scope | Blocking behavior |
| --- | --- | --- |
| `govulncheck` | Reachable vulnerabilities in Repora and Go dependencies | Any reported reachable vulnerability blocks the workflow until fixed or explicitly reviewed. |
| CodeQL `security-extended` | Go source and supported data-flow queries | Analysis failures block CI. Findings are reviewed in GitHub code scanning; unresolved high-confidence findings block release through the release checklist. |
| Gitleaks | Git history available in the CI checkout | Any unignored finding blocks CI. Logs use redaction and must not reproduce secret values. |
| `go-licenses check` | Dependencies of the shipped `repoctl` command | Forbidden or unknown dependency licenses block CI. |
| `go-licenses report` | Dependency name, versioned license URL, and detected license | Produces a retained CSV inventory for review; an empty report is a failure. |
| Workflow policy | GitHub Actions syntax, pins, permissions, timeouts, and triggers | Any policy violation blocks CI. |

The security workflow runs on pull requests, pushes to `main`, a weekly schedule, and manual dispatch. Pull-request jobs receive read-only repository permissions and no repository secrets.

## Local reproduction

```bash
make security-secrets
make security-licenses
```

`security-secrets` scans the Git history available in the checkout. Use a full clone when reproducing a historical finding.

`security-licenses` checks the dependencies used by `./cmd/repoctl`, ignores Repora's own BSL-licensed packages, and writes `artifacts/security/licenses.csv`.

## Triage

For every finding, record:

- tool and rule or advisory identifier;
- affected file, package, dependency, or workflow;
- whether the finding is reachable or exploitable in Repora's supported operating model;
- impact and release relevance;
- remediation, accepted residual risk, or false-positive rationale;
- owner and review trigger;
- links to the issue, pull request, advisory, or code-scanning result.

Do not paste credentials, tokens, private keys, authenticated URLs, or unredacted scanner output into issues or pull requests.

## Suppression rules

A suppression is acceptable only when all of the following are true:

1. the finding has been reproduced and understood;
2. the matched value is demonstrably non-secret or the vulnerable path is demonstrably unreachable;
3. the suppression is as narrow as the tool permits;
4. the reason is repository-visible and reviewable;
5. a concrete trigger exists for revisiting it.

For Gitleaks, prefer a line-level `gitleaks:allow` annotation for an intentional test fixture. Use `.gitleaksignore` only for a stable fingerprint after reviewing the exact finding. Never add a broad path or rule exclusion merely to make CI pass.

For dependency or CodeQL findings, prefer remediation or version updates. When temporary acceptance is necessary, retain the finding and document release disposition rather than weakening the scanner globally.

## Severity and release decisions

Priority, severity, and release blocking are separate dimensions:

- an active exposed credential is P0 and requires rotation plus history/remediation review;
- a reachable critical or high dependency vulnerability is normally P0/P1 and blocks release;
- a high-confidence CodeQL finding that affects supported input or mutation paths blocks release;
- lower-confidence or unreachable findings require documented triage but need not block unrelated development;
- unknown or forbidden dependency licensing blocks redistribution until resolved.

The release checklist is the final accountable gate for reviewed security findings. CI must remain low-noise and fail on objective scanner or policy violations; human risk acceptance must be explicit rather than encoded as a silent workflow exception.
