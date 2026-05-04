# ADR-0010: Unified Diff Model

Status: Draft

## Decision

Repora shall evolve toward a unified diff model capable of representing
differences across multiple domains, including Git refs, file content,
workflows, and artifacts, using a common abstraction layer.

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

## Domain Mapping

### Refs (v0.1)

```text
identity: <remote>/<ref>
diff: commit graph relationship (ahead/behind/diverged)
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

## Rationale

- Enables a single reconciliation engine across heterogeneous domains
- Avoids duplicating diff logic for refs vs files vs workflows
- Supports incremental expansion without redesigning the core planner

## Consequences

- Requires domain-specific adapters for diff generation
- Increases abstraction complexity
- Provides long-term extensibility and consistency
