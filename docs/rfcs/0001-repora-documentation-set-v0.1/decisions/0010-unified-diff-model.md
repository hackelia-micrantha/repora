# ADR-0010: Unified Diff Model

Status: Draft

## Decision

Repora shall evolve toward a unified diff model capable of representing
differences across multiple domains, including Git refs, file content,
workflows, and artifacts, using a common abstraction layer.

The first implementation slice defines a narrow, versioned reconciliation plan
artifact for Git ref updates. It is an internal durable representation and does
not yet replace the existing CLI plan or apply output.

## Core Abstraction

All differences are represented as state objects with the following structure:

```text
StateObject {
  domain: refs | files | workflows | artifacts
  identity: string
  desired: any
  observed: any
  diff: structured delta
  state: EQUAL | DRIFT | DIVERGED | UNKNOWN
}
```

## Plan Artifact v1

The initial serialized envelope is:

```text
Artifact {
  version: 1
  kind: repora.io/reconciliation-plan
  repositories: Repository[]
}

Repository {
  uid: durable repository identity
  id: human-facing repository identity
  actions: Action[]
}

Action {
  type: PUSH_BRANCH
  source: provider, remote name, branch
  target: provider, remote name, branch
  diff: observed target OID -> desired source OID
  force: bool
  reason: string
}
```

Repository and action order are preserved. Struct-backed JSON encoding makes
repeated serialization deterministic for identical ordered inputs.

The artifact excludes resolved transport URLs, credentials, and local checkout
paths. Validation rejects unsupported versions, kinds, action types, missing
identity or safety fields, unknown JSON fields, and values that look like URLs,
credentials, or absolute local paths.

## Domain Mapping

### Refs (v0.1)

```text
identity: <remote>/<ref>
diff: observed target OID -> desired source OID
```

### Files (future)

```text
identity: path/to/file
diff: textual or semantic diff
```

### Workflows (future)

```text
identity: workflow identifier
diff: structural YAML/JSON diff
```

### Artifacts (future)

```text
identity: registry/image:tag or model version
diff: version mismatch or missing artifact
```

## Diff Classification

All domains normalize into:

- `EQUAL`
- `DRIFT` (unidirectional difference, safe to reconcile)
- `DIVERGED` (bidirectional difference, requires policy/force)
- `UNKNOWN` (unable to determine)

## Planner Integration

The planner shall operate over state objects, producing actions such as:

```text
Action {
  type: SYNC | UPDATE | CREATE | DELETE
  domain: refs | files | workflows | artifacts
  target: identity
  destructive: bool
}
```

For the current ref-only slice, the artifact converts losslessly to and from the
existing in-memory `plan.ReconciliationPlan`. Executor consumption of the
validated artifact is a follow-up slice.

## Rationale

- Enables a single reconciliation engine across heterogeneous domains
- Avoids duplicating diff logic for refs vs files vs workflows
- Supports incremental expansion without redesigning the core planner
- Creates a reviewable and testable boundary before changing executor or CLI behavior

## Consequences

- Requires domain-specific adapters for diff generation
- Increases abstraction complexity
- Provides long-term extensibility and consistency
- Version and kind changes require explicit compatibility handling
- Artifact safety validation is intentionally stricter than arbitrary Git metadata
