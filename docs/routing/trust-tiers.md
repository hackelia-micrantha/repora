# Document routing trust tiers

Status: Current

## Purpose

Document routing is part of Repora's AI trust boundary. Path matching and keyword classification decide relevance; trust tiers decide whether a matched source is eligible by default and how strongly it should influence an AI-assisted operation.

Trust metadata is explicit configuration. Repora does not infer authority from writing style, file age, popularity, or model judgment.

## Supported tiers

| Tier | Meaning | Default behavior |
| --- | --- | --- |
| `canonical` | Current source of truth such as root documentation, requirements, decisions, schemas, and router configuration | Included by default |
| `implementation` | Source code, tests, build scripts, and module metadata that reflect executable behavior | Included by default |
| `generated` | Build output, reports, generated documentation, and derived artifacts | Excluded unless explicitly requested |
| `experimental` | Drafts, spikes, prototypes, and unaccepted design work | Excluded unless explicitly requested |
| `archived` | Historical material retained for traceability rather than current authority | Excluded unless explicitly requested |
| `external` | Vendored, imported, or third-party content outside repository authority | Excluded unless explicitly requested |

Unclassified content is excluded. This fails conservatively when a new path has not yet been assigned an authority boundary.

## Selection order

Routing applies these stages in order:

1. Match one or more routes from the normalized query.
2. Expand route include and exclude patterns.
3. Classify candidate paths using the first matching trust rule in configuration order.
4. Remove candidates whose tier is not eligible for the operation.
5. Apply deterministic path ordering, deduplication, and route budgets.
6. Record trust decisions in a future context receipt.

Trust does not make irrelevant content relevant. A canonical file must still match the selected route. Likewise, an explicitly requested archived file remains archived; explicit inclusion changes eligibility, not its authority label.

## Explicit inclusion

Generated, experimental, archived, and external content may be included only when the caller or route explicitly requests the tier. Consumers must surface that weaker authority in their output and future context receipts.

Explicit inclusion must be narrow. Enabling an entire excluded tier globally merely to make a query return more files defeats the trust boundary.

## Precedence and overlap

Trust rules are evaluated in declared order and the first matching rule wins. More specific paths therefore belong before broad globs. Conflicting or duplicate path classifications should fail validation once route schema validation is promoted into the runtime.

Examples:

- `docs/archive/**` must classify as `archived` before the broad `docs/**` canonical rule is applied.
- `artifacts/**` remains generated even if it contains copied Markdown.
- vendored source remains external rather than implementation owned by Repora.

## Security rationale

Conservative trust defaults reduce these failure modes:

- archived requirements overriding current decisions;
- generated reports being mistaken for source evidence;
- experimental prompts influencing production changes;
- imported documentation carrying prompt-injection instructions;
- newly added, unclassified paths silently entering the retrieval boundary.

Trust tiers are not authenticity verification. Canonical content can still be wrong or maliciously modified, and implementation content can drift from intended policy. Reviewers must continue to compare claims against code, tests, schemas, decisions, and repository history.

## Ownership and maintenance

Changes to trust tiers or path rules are security-sensitive routing-policy changes. Pull requests must explain:

- which paths change authority;
- why default eligibility changes;
- whether lower-trust content can now influence mutation planning;
- which deterministic route fixtures prove precedence and exclusion behavior.

Future subsystem manifests may contribute trust metadata only through explicit top-level composition. Automatic discovery and automatic trust scoring remain out of scope.

## Relationship to context receipts

Issue #17 will make trust decisions inspectable by recording selected and excluded paths, assigned tiers, explicit-inclusion reasons, and the applicable policy version. This document defines the current taxonomy and default behavior; it does not introduce a provenance database.
