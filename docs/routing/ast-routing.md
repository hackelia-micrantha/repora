# Go AST-aware source routing

Status: Current

## Purpose

AST-aware routing refines an already selected route when implementation context needs symbol-, package-, command-, or file-level precision. It does not replace route-first retrieval and does not use embeddings or semantic similarity.

## Initial language boundary

The first implementation supports Go only. Source is parsed with the Go standard library (`go/parser` and `go/ast`), so no language server, external index service, or generated database is required.

Only non-test `.go` files in repository-owned source trees are indexed. `.git`, `vendor`, `bin`, `dist`, and `artifacts` are excluded from AST indexing.

## Selector contract

Routes may declare:

```yaml
source_selectors:
  language: go
  packages:
    - internal/plan
  symbols:
    - Reconcile
    - ReconciliationPlan
  commands:
    - repoctl
  files:
    - internal/plan/*.go
```

Supported selectors are:

- `packages`: repository-relative Go package directories;
- `symbols`: exported Go declarations (functions, types, constants, or variables);
- `commands`: direct `cmd/<name>` command directories;
- `files`: repository-relative file globs.

Within one selector family values are ORed. Across configured selector families the match is ANDed. For example, `packages: [internal/plan]` plus `symbols: [Reconcile, ReconciliationPlan]` selects files in `internal/plan` that define at least one of those exported symbols.

## Composition with route-first selection

1. Normal query matching chooses one or more routes.
2. Route `include` and `exclude` patterns establish the candidate boundary.
3. Trust-tier eligibility is applied.
4. If `source_selectors` exists, deterministic AST selectors refine implementation candidates inside that boundary.
5. Selected paths are sorted lexicographically and remain subject to the route budget.
6. Match reasons are recorded as stable strings such as `package:internal/plan`, `symbol:Reconcile`, `command:repoctl`, and `file:cmd/repoctl/*.go`.

Selectors must never expand outside the route include boundary or bypass trust exclusions.

## Determinism and evidence

The checked-in AST fixture runner parses repository source directly and asserts exact paths plus exact match reasons. It does not persist an index. A future runtime implementation may cache an index, but cache contents must be reproducible from the same repository revision and selector contract.

Context receipts should record selector reasons alongside the selected path and existing trust evidence. Receipts remain evidence, not executable routing authority.

## Security boundaries

- AST parsing is local and read-only.
- Vendored and generated trees are not indexed by default.
- Only exported declarations are symbol-addressable in this first slice.
- No source code is executed during indexing.
- File selectors remain repository-relative.
- AST selectors cannot override trust tiers or route exclusions.

## Deferred work

- multi-language parsing;
- language-server integration;
- inferred ownership;
- call-graph or dependency-graph ranking;
- fuzzy symbol matching;
- embeddings or vector search;
- persistent/shared AST indexes.

## Validation

Run:

```sh
make route-test
```

The Go AST fixture contract validates representative package, exported-symbol, command, and file selectors with deterministic ordering and inspectable reasons.
