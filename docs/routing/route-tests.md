# Deterministic route tests

Status: Current

Repora validates `.repora/document-router.yaml` against checked-in query fixtures before changes merge. The tests protect the existing declarative routing contract without introducing a network service, embeddings, fuzzy scoring, or a complete interactive router runtime.

## Run locally

```sh
make route-test
```

The command uses the repository Go toolchain and existing `gopkg.in/yaml.v3` dependency. It requires no external service and performs no repository mutation.

`make check` includes this validation, so the same fixtures run in pull-request CI and the scheduled Go compatibility jobs.

## Fixture contract

Fixtures live at `.repora/route-tests.json`:

```json
{
  "version": 1,
  "kind": "document-route-tests",
  "cases": [
    {
      "name": "architecture class",
      "query": "Explain planner and executor architecture",
      "expect_routes": ["architecture-core"],
      "expect_include": ["README.md", "docs/architecture/**"],
      "expect_exclude": ["tests/**"],
      "expect_budget": {
        "max_files": 10,
        "max_bytes": 120000,
        "max_tokens_hint": 12000
      }
    }
  ]
}
```

Each case has:

- a unique human-readable `name`;
- a representative `query`;
- the complete ordered `expect_routes` result;
- optional first-route or fallback include/exclude assertions;
- optional exact budget assertions;
- optional `expect_fallback` when no route matches.

Unknown fixture fields fail parsing. Duplicate names and unsupported contract versions fail validation.

## Matching and ordering contract

The fixture runner models the current deterministic classifier boundary:

1. Collapse query and route-keyword whitespace.
2. Compare lower-cased text.
3. Select a route when the normalized query contains one of its explicit `when.any_of` terms.
4. Sort matched routes by descending `priority`.
5. Break equal-priority ties by lexicographic route ID.
6. When no route matches, select the first declared fallback.

This is intentionally literal and explainable. It does not claim semantic relevance beyond the explicit route configuration.

## Coverage boundary

The initial fixtures cover:

- every existing route class;
- first-route include and exclude behavior;
- exact route and fallback budgets;
- fallback selection;
- priority ordering across multiple matches;
- stable lexicographic tie-breaking.

Future route features must extend these fixtures in the same change:

- subsystem composition under issue #9;
- trust-tier precedence under issue #23;
- summary-first expansion under issue #20;
- context receipt assertions under issue #17;
- AST selectors under issue #14.

Those capabilities are not implemented or implied by this test runner.

## Maintenance

A fixture change is a routing-policy change and should be reviewed together with the corresponding router configuration change. Do not weaken an expectation merely to accept accidental drift. When intent changes, update the route and fixture together and explain the changed selection boundary in the pull request.
