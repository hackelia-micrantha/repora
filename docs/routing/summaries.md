# Hierarchical routing summaries

Status: Current

## Purpose

Repora uses small `SUMMARY.md` files at selected repository boundaries to reduce routed-context cost. A summary is an orientation layer: it points to authoritative material and explains when deeper retrieval is required.

A summary must not become a parallel source of truth.

## Trust classification

`SUMMARY.md` files are classified as `generated` because they are derived from canonical documents and implementation evidence. They are excluded by the normal trust policy unless a route explicitly opts into the `generated` tier for summary-first retrieval.

Explicit inclusion changes eligibility only. It does not relabel a summary as canonical.

## Required structure

Each maintained summary contains:

1. **Purpose** — what the subtree owns.
2. **Canonical sources** — links to the smallest authoritative documents.
3. **Ownership boundaries** — what this subtree does and does not define.
4. **Expand when** — deterministic reasons to load deeper material.
5. **Exclusions and stale areas** — historical, generated, experimental, or otherwise non-authoritative material that should not be used silently.

Keep summaries concise. They should link to details rather than reproduce requirements, schemas, algorithms, or long design explanations.

## Progressive retrieval

A route that uses a summary declares:

- `summary_first`: ordered summary paths to load before deeper candidates;
- `trust_include: [generated]`: the explicit trust opt-in required for those summaries;
- normal `include` patterns for deeper material.

The expected flow is:

1. select the route deterministically;
2. load the route's `summary_first` files;
3. stop when the summary answers an orientation-level question;
4. expand into canonical or implementation sources when the question needs details, verification, exact contracts, or mutation planning;
5. record both summary selection and deeper expansion in a context receipt when a receipt is produced.

Expansion is required when:

- the caller asks for exact behavior, compatibility, security, or failure semantics;
- a summary points to a canonical source for the requested detail;
- implementation and documentation need to be compared;
- a plan, recommendation, or mutation depends on the claim;
- the summary is ambiguous, incomplete, or potentially stale.

## Maintenance

Summary changes should accompany the canonical change they describe when practical. Reviewers should reject summaries that duplicate detailed requirements or make claims not supported by their linked sources.

Automatic summary generation, external-content summarization, semantic/vector retrieval, and one-summary-per-file are out of scope for this contract.
