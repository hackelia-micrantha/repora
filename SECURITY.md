# Security Policy

## Project status

Repora is pre-alpha. Security-relevant behavior, compatibility contracts, and release controls are still evolving, and no long-term support window is currently promised for historical development snapshots.

Security fixes are developed on the current `main` branch and included in subsequent releases as appropriate. The latest published release and current source may therefore differ while pre-alpha development is active.

## Reporting a vulnerability

Do not publish exploit details, credentials, private repository data, or other sensitive evidence in a public issue.

Prefer GitHub's private vulnerability-reporting or security-advisory mechanism for this repository when it is available to you. If no private GitHub reporting path is available, open a public issue containing only a request for a private maintainer contact channel and enough non-sensitive context to route the report. Do not include vulnerability details in that public issue.

Include, where possible:

- the affected Repora version, tag, or commit;
- the affected command, artifact, workflow, or trust boundary;
- concise reproduction conditions without real credentials or private data;
- expected impact;
- any known workaround or containment step.

No response-time or remediation SLA is currently promised.

## Security scope

Reports are especially useful when they concern:

- credential or token disclosure;
- unintended repository or provider mutation;
- stale-plan, lease, or authorization bypasses;
- path traversal, symlink, cache, journal, or template-boundary escapes;
- malformed or untrusted repository data causing unsafe execution or sensitive-data reflection;
- CI/CD permission, runner, release, or supply-chain weaknesses in Repora-maintained workflows;
- integrity failures in versioned plans, results, evidence, posture artifacts, or release packages.

Expected limitations documented in the README, denial-of-service requiring unreasonable local resource exhaustion, and vulnerabilities solely in unsupported historical snapshots may be handled as normal maintenance rather than security defects.

## Handling sensitive evidence

Use synthetic credentials and disposable repositories whenever possible. Redact tokens, private URLs, local usernames, filesystem paths, proprietary source, and unrelated repository content from logs or examples.

Repora's security model and automated validation boundaries are documented in [`README.md`](README.md), [`docs/architecture/current-system.md`](docs/architecture/current-system.md), and [`docs/security-ci.md`](docs/security-ci.md).
