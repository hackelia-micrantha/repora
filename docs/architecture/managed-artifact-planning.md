# Managed artifact planning primitives

Status: Current

## Scope

This document describes the implemented pure planning layer for managed README artifacts: contained local template loading, deterministic rendering, exact observed-state modeling, byte-aware review diff construction, and managed-artifact plan assembly.

The planner currently depends on a `READMEObserver` interface. A production Git-backed observer, CLI entry point, dry-run/apply preflight, commit creation, and remote push are **not** implemented by this layer and remain owned by issue #12.

## Input boundary

`BuildPlan(configPath, spec, observer)` processes only repositories with explicit `artifacts.readme` configuration.

- If no repository is configured for README management, it returns an explicit empty managed-artifact plan without requiring an observer or touching template paths.
- Configured repositories are sorted by durable UID, then ID, before template or observer work so plan ordering is deterministic.
- Repository ID/UID and canonical provider/path must already be canonical plan-safe identity. The planner fails before template I/O when those fields require normalization or contain unsafe display/path syntax.

The normal CLI configuration loader remains responsible for broader repository topology validation. The planner repeats the identity checks needed for its own durable artifact boundary so direct callers cannot bypass them.

## Template loading

`LoadTemplate` resolves the physical configuration file first and treats its containing directory as the template root.

A template reference must be:

- relative and canonical;
- slash-separated and portable;
- free of absolute paths, URLs, `~`, backslashes, traversal, redundant segments, controls, and surrounding whitespace.

Symlinks may resolve within the configuration root. A template that resolves outside the root is rejected. The resolved target must be a regular file and is read through a 1 MiB bound.

The loader returns exact template bytes. Rendering performs text validation and line-ending normalization; the managed plan records SHA-256 of the exact loaded template bytes.

## Observation interface

The pure planner consumes:

```text
READMEObservation {
  branch
  base_oid
  present
  mode
  raw content bytes
}
```

For a present README, mode must be `100644` or `100755`, content must be bounded safe UTF-8 text, and the raw bytes are preserved for SHA-256 and review generation. CRLF is allowed in observed text so a line-ending-only change remains visible.

For a missing README, mode and content must be absent/empty. Missing and present-empty are distinct states.

The observer is responsible for proving that the observation came from the exact canonical default-branch tree. The upcoming Git-backed observer slice will bind these fields to Repora's existing Git cache/fetch layer.

## Byte-aware review diff

`ReviewDiff` is a deterministic review representation, not an applyable patch.

It uses fixed envelope labels:

```text
--- a/README.md
+++ b/README.md
@@ ...
```

Content is represented as JSON-quoted exact byte-backed strings. This keeps control bytes out of terminal output while making line terminators explicit. For example, a CRLF-to-LF-only change is visible as:

```text
-"# Title\r\n"
+"# Title\n"
```

A missing README and a present zero-byte README are also distinct. Creating a zero-byte README includes:

```text
+""
```

The diff finds exact common prefix/suffix line segments, emits at most three context lines on each side, and JSON-quotes the changed before/after block as one string each. Grouping the changed block bounds JSON-escaping amplification while keeping the existing managed-plan review ceiling sufficient for 1 MiB README inputs.

Review text inherits the managed-text safety policy: invalid UTF-8, terminal controls, Unicode format/bidirectional controls, and Unicode line/paragraph separators fail closed.

## Plan construction

For each configured repository, the builder:

1. validates durable target identity;
2. loads the bounded contained template;
3. renders desired README bytes deterministically;
4. requests exact README observation;
5. validates observed branch/base OID/mode/content;
6. omits the repository when a present README already equals desired bytes;
7. otherwise computes the byte-aware review diff;
8. preserves an existing regular-file mode or uses `100644` for creation;
9. records observed raw-content SHA-256, desired SHA-256/content, exact template SHA-256, target identity, and base OID;
10. validates the completed `repora.io/managed-artifact-plan` v1 before returning it.

The plan contains no local template path, cache path, credentials, timestamp, author identity, environment value, or Git command line.

## Still deferred

Issue #12 still requires:

- Git-backed canonical README observation;
- user-facing plan/review CLI behavior;
- exact-plan stale preflight and dry-run;
- isolated commit creation that changes only root `README.md`;
- guarded canonical push;
- execution result/evidence output;
- fresh mirror reconciliation after a successful canonical README change.
