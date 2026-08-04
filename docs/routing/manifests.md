# Document route manifests

Status: Current

## Purpose

Subsystem manifests let repository areas own route metadata without turning `.repora/document-router.yaml` into a central dumping ground.

The root router remains the only composition root. Manifests are never discovered recursively or implicitly.

## Root composition

The root router lists manifests in deterministic order:

```yaml
manifests:
  - docs/routing/router.manifest.yaml
```

Each path must be repository-relative, remain inside the repository, and identify a checked-in YAML file. Absolute paths, parent traversal, and duplicate manifest references fail validation.

## Manifest contract

```yaml
version: 1
kind: document-route-manifest
owner: routing

routes:
  - id: prompts-and-routing
    class: prompts
    priority: 95
    when:
      any_of:
        - document router
    include:
      - docs/routing/**
    exclude:
      - tests/**
    budget:
      max_files: 8
      max_bytes: 90000
      max_tokens_hint: 9000
```

Required fields:

- `version: 1`
- `kind: document-route-manifest`
- non-empty `owner`
- at least one route
- globally unique, non-empty route IDs

## Composition semantics

1. Load root routes in declaration order.
2. Load manifests in the exact order listed by the root router.
3. Append each manifest's routes in declaration order.
4. Reject duplicate route IDs across the root and every manifest.
5. Apply the existing priority and route-ID tie-breaking rules during query matching.

Manifest order does not override route priority. It only makes composition deterministic and reviewable.

## Ownership and trust

Manifest owners maintain route intent, include/exclude patterns, and budgets for their subsystem. Trust tiers remain centrally defined by the root router so a subsystem cannot silently weaken repository-wide source eligibility.

Future manifest trust metadata may only compose through an explicit, validated root contract. Automatic trust scoring and automatic manifest discovery remain out of scope.

## Validation

Run:

```bash
make route-test
```

The validation contract checks root and manifest versions, safe manifest paths, unique manifest references, required owners, non-empty manifests, globally unique route IDs, and the existing deterministic query fixtures.

## Security boundary

Explicit composition prevents:

- hidden manifests being loaded from unrelated directories;
- recursive discovery loops;
- path traversal outside the repository;
- duplicate route IDs silently replacing policy;
- subsystem files weakening central trust defaults.

Remote manifests, executable manifest logic, semantic search, and runtime plugin discovery are not supported.
