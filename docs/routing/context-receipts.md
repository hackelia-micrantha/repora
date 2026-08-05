# Context receipts

Status: Current

## Purpose

A context receipt is a versioned, deterministic evidence artifact describing how Repora selected routed context for an AI-assisted operation.

Receipts make routing decisions inspectable and challengeable. They are not executable authority, a durable provenance database, or proof that selected content was correct.

## Contract

The v1 JSON schema is `schemas/context-receipt-v1.schema.json`. A complete example is available at `examples/context-receipt-v1.json`.

A receipt records:

- the normalized query;
- repository revision and dirty state;
- router policy version and SHA-256 digest;
- selected route IDs and match terms, or the selected fallback;
- explicitly composed manifests;
- selected paths, trust tiers, reasons, sizes, content hashes, and bounded snippet hashes;
- excluded paths or patterns and exclusion reasons;
- configured and consumed budgets.

## Routing-policy digest

`routing.policy_sha256` is the digest of the canonical fully composed routing policy used for the decision. It covers the root router, explicitly listed subsystem manifests, trust configuration, route ordering, fallbacks, and selection settings after validation.

Manifest paths remain in the receipt so a reviewer can identify the composition inputs. The composed digest is the binding evidence; a digest of only the root YAML would be insufficient because a manifest could change independently.

## Route and fallback evidence

A receipt records exactly one routing mode:

- one or more selected routes, with their deterministic order and normalized match terms; or
- one selected fallback ID when no route matched.

Routes and a fallback cannot both be selected. This preserves evidence for unmatched queries without pretending a fallback was a normal route match.

## Determinism

For the same normalized query, repository revision, composed routing policy, trust policy, manifest set, and candidate inventory, receipt serialization must be byte-for-byte deterministic.

Producers must:

1. normalize query whitespace without rewriting meaning;
2. sort match terms and manifest paths lexicographically;
3. sort selected inputs by repository-relative path;
4. sort snippet entries by index;
5. sort excluded inputs by path and reason;
6. preserve selected route order after priority and route-ID tie-breaking;
7. serialize JSON with lexicographically sorted object keys, two-space indentation, UTF-8, and one trailing newline;
8. use lowercase hexadecimal SHA-256 values.

Wall-clock timestamps and random identifiers are intentionally absent from v1 because they would make equivalent decisions serialize differently.

## Budget integrity

Consumed budget fields are evidence, not estimates copied without validation:

- `selected_files` must equal the selected input count;
- `selected_bytes` must equal the sum of selected input byte counts;
- selected files, bytes, and estimated tokens must remain within configured limits;
- total snippet bytes for an input cannot exceed that input's byte count.

The token count remains an estimate, but it must use the producer's documented tokenizer or estimation method consistently.

## Replay and challenge

A receipt supports replay or challenge by identifying:

- the repository revision;
- the exact composed routing-policy digest;
- route or fallback identity and manifest identities;
- content and snippet hashes;
- trust classifications;
- selection and exclusion reasons;
- budget limits and consumption.

A consumer can compare these values against a checkout and routing configuration. A receipt does not authorize a new retrieval or mutation and must not be treated as a reusable plan.

## Content and redaction boundary

Receipts store metadata and hashes by default, not source bodies. Snippet entries record index, byte count, and SHA-256 only.

Receipt producers must not persist:

- credentials, bearer tokens, private keys, cookies, or authentication headers;
- credential-bearing or tokenized URLs;
- absolute local filesystem paths or backslash-based platform paths;
- full source files or unbounded snippets;
- unrelated repository content.

Paths are repository-relative POSIX paths. Individual snippet sizes are bounded to 4096 bytes by the schema even when future producers optionally attach separately managed snippet material.

Automated redaction checks are defense in depth, not proof that arbitrary text is secret-free. Producers remain responsible for excluding sensitive query text, reasons, and path metadata before receipt creation.

## Retention

Receipts are local operator-managed artifacts. The recommended default location is `.repora/receipts/`, excluded from version control unless a receipt is deliberately curated as test or review evidence.

Retention policy belongs to the operator. Repora does not provide a long-term receipt database, automatic upload, or remote retention service.

## Relationship to other artifacts

- **Context receipt:** explains routed evidence selection.
- **Reconciliation plan:** describes reviewed Git effects.
- **Diff or proposal:** describes a candidate content change.
- **Execution journal:** records mutation intent and outcomes.

A later workflow may link these artifacts by digest, but none inherits execution authority from another.

## Validation

Run:

```bash
make receipt-test
```

The test contract validates the curated example and negative cases for non-canonical serialization, credential-like content, credential-bearing URLs, unsafe paths, over-large snippets, inconsistent budgets, duplicate snippet indices, and invalid route/fallback combinations.

## Dependency order

The receipt contract depends on deterministic route tests (#18), trust tiers (#23), and explicit manifest composition (#9), all of which now exist.

Hierarchical summaries (#20) may later appear as selected inputs with their own trust tier and content hash. AST-aware selectors (#14) may extend snippet metadata in a future receipt version rather than changing v1 semantics.

Anthesis integration and a general provenance system remain out of scope.
