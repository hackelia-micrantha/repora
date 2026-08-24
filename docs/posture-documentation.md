# Documentation posture v1

Status: Current

`repoctl posture docs` is Repora's deterministic documentation/README hygiene fact collector. It extends the posture evidence layer established by `repora.posture-inventory` v1 without changing that existing contract.

The command is read-only and emits observations only. It does **not** assign severity, create findings, judge prose quality, rewrite documentation, open pull requests, or mutate provider/repository state. Policy evaluation remains owned by the later posture-policy layer.

## Command

```bash
repoctl posture docs OWNER/REPO > documentation-posture.json
```

Public GitHub repositories need no token. Private repositories or provider evidence requiring authenticated read access may use `GITHUB_TOKEN` or `GH_TOKEN` from the environment.

The command emits `repora.posture-documentation` v1 JSON. The collector uses the same `observed`, `unknown`, and `unavailable` fact semantics as the GitHub repository/CI inventory.

## Observation profile

A repository may declare documentation observation targets in:

```text
.repora/posture-documentation.yaml
```

The versioned profile contract is `repora.posture-documentation-profile` v1.

Example:

```yaml
kind: repora.posture-documentation-profile
version: 1
name: service

documents:
  - README.md
  - SECURITY.md
  - docs/architecture.md

readme:
  path: README.md
  sections:
    - Overview
    - Security
  links:
    - SECURITY.md
    - docs/architecture.md

content_markers:
  - id: current-toolchain
    path: docs/ci.md
    contains: Go 1.25.13
```

The profile says **what to observe**, not how to score it. A missing configured document/section/link/marker is an observed `false` fact when the repository evidence is complete. A later policy may decide that one repository profile requires a fact while another only recommends it.

If the profile is known to be absent from a complete Git tree, Repora uses a conservative built-in `baseline` profile that observes only root `README.md`. If the Git tree is truncated and the profile is not visible, Repora does **not** assume it is absent; profile and README facts remain `unknown`.

Profile paths must be normalized repository-relative paths. The first contract is intentionally bounded to at most 256 combined document, README section/link, and content-marker targets.

## Documentation facts

The v1 output records:

- default branch and immutable observed commit SHA;
- whether a repository-declared documentation observation profile exists;
- active profile name;
- configured document presence;
- configured README section presence;
- configured README repository-link presence;
- configured exact content-marker presence;
- document-router metadata presence;
- whether the router's **trust metadata** is usable by this bounded collector;
- trust tier for observed documentation targets when routing trust metadata is usable;
- source/reference evidence for each fact.

`routing_trust_metadata_usable` is deliberately narrower than “the complete document router is valid.” Documentation posture parses only the trust subset it needs for authority classification. Full document-router validity remains owned by the routing contract/validators.

README section matching is deterministic and case-insensitive over ATX Markdown headings (`#` through `######`). Fenced code blocks are ignored so example headings do not satisfy section observations.

README link matching considers inline repository-relative Markdown links and resolves them relative to the README path. External URLs, mail links, fragment-only links, and paths escaping the repository are not treated as repository-document links.

## Content markers and stale metadata

`content_markers` provide a narrow deterministic mechanism for metadata that should stay synchronized across files, such as a documented toolchain version or canonical command name.

The collector performs an exact substring observation. It does not interpret semantics or decide whether a mismatch is acceptable. The output stores only the lowercase SHA-256 digest of the configured expected marker plus the boolean/unknown/unavailable fact; the configured marker text is not copied into the output artifact.

This keeps stale-metadata observation explicit and profile-driven while avoiding an LLM or prose-quality boundary.

## Routing and canonical documents

If `.repora/document-router.yaml` exists, documentation posture reads the router's explicit trust rules to classify configured document paths. The collector preserves the existing routing trust model:

- `canonical`
- `implementation`
- `generated`
- `experimental`
- `archived`
- `external`
- `unclassified` when no trust rule matches

When patterns overlap, the greatest literal specificity wins, then pattern length, then declaration order. This matches the existing routing trust-policy rule, so a narrow `generated` or `archived` pattern cannot be shadowed by a broad canonical pattern.

Explicit inclusion never changes a document's authority label. A generated or archived document remains generated/archived and is never silently promoted to canonical by documentation posture.

The v1 posture reader supports the wildcard forms used by Repora's current trust rules (`*`, `**`, and `?`). A trust rule using unsupported character-class pattern syntax is reported as unusable for posture classification rather than being interpreted optimistically. This does not assert that the complete document-router file is invalid for every other consumer.

## Evidence and failure semantics

The collector reads repository metadata, the default branch, one recursive Git tree, and only the blobs needed for the active profile/router. All provider access goes through the same GET-only `GitHubReader` capability used by repository/CI posture.

Evidence states remain conservative:

- complete tree + missing path -> observed `false`;
- truncated tree + missing path -> `unknown`;
- provider-hidden blob/tree -> `unavailable`;
- malformed/unsupported declared profile -> profile name and dependent facts `unknown`;
- malformed/unsupported routing trust subset -> `routing_trust_metadata_usable` observed `false`, document trust tiers `unknown`;
- document larger than the 2 MiB normalization bound -> dependent content facts `unknown`.

Unexpected transport failures and malformed provider responses remain operational errors and cause a nonzero command exit.

## Security and trust boundary

The profile, router, and inspected Markdown are untrusted repository data. Repora does not execute code, commands, templates, hooks, or embedded Markdown/HTML while collecting documentation posture.

Configured exact marker text is not serialized in the output. Malformed profile/router evidence uses generic diagnostics rather than embedding raw parser errors or repository content, preventing a malformed repository-controlled file from reflecting sensitive configured values into posture artifacts.

The repository-declared profile is intentionally not a security policy. A repository can influence which documentation signals it asks Repora to observe, but it cannot use the profile to assign severity, suppress external policy, grant mutation authority, or mark generated/archived content canonical outside the explicit routing trust rules.

## Relationship to posture policy

`repora.posture-documentation` v1 owns deterministic documentation facts. The offline posture policy layer consumes this artifact for expected-vs-observed findings, explicit severity, remediation guidance, exceptions, and deterministic reports.

Policy consumes this artifact rather than re-reading README/docs independently, preserving one observation/provenance boundary. Any future issue/PR remediation remains a separate mutation capability.
