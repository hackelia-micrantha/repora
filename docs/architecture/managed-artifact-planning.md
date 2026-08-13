# Managed artifact planning primitives

Status: Current

## Scope

This document describes the implemented read-only planning layer for managed README artifacts: contained local template loading, deterministic rendering, exact canonical Git-tree observation, byte-aware review diff construction, managed-artifact plan assembly, and the user-facing `plan-readme` review command.

The planner depends on a `READMEObserver` interface. `NewGitREADMEObserver` provides the production Git-backed implementation. Dry-run/apply preflight, commit creation, and remote push are **not** implemented by this layer and remain owned by issue #12.

## Input boundary

`BuildPlan(configPath, spec, observer)` processes only repositories with explicit `artifacts.readme` configuration.

- If no repository is configured for README management, it returns an explicit empty managed-artifact plan without requiring an observer or touching template paths.
- All configured repository IDs, durable UIDs, canonical providers, and provider-relative paths are validated before template or observer I/O.
- Duplicate managed UIDs, IDs, or canonical provider/path identities fail before I/O.
- Configured repositories are sorted by durable UID, then ID, before template or observer work so plan ordering is deterministic.
- Managed repositories must use provider/path canonical identity and must not retain a legacy canonical URL.

The normal CLI configuration loader remains responsible for broader repository topology validation. The planner repeats the identity checks needed for its own durable artifact boundary so direct callers cannot bypass them.

## Template loading

`LoadTemplate` resolves the physical configuration file first and treats its containing directory as the template root. The configuration path itself must resolve to a regular file.

A template reference must be:

- relative and canonical;
- slash-separated and portable;
- free of absolute paths, URLs, `~`, backslashes, traversal, redundant segments, unsafe controls, and surrounding whitespace.

Symlinks may resolve within the configuration root. A template that resolves outside the root is rejected. The resolved target must be a regular file and is read through a 1 MiB bound.

The loader returns exact template bytes. Rendering performs text validation and line-ending normalization; the managed plan records SHA-256 of the exact loaded template bytes.

## Observation interface

The planner consumes:

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

For a missing README, mode and content must be absent/empty. Missing and present-empty are distinct states. The observed branch must satisfy the same symbolic-ref validation as the durable plan contract, and the base OID must be an exact 40- or 64-character hexadecimal object ID.

### Production Git-backed observer

`NewGitREADMEObserver` binds the observation to Repora's existing bare-cache and HTTPS transport model.

For each managed repository it:

1. re-validates the durable ID and provider/path canonical identity;
2. resolves the canonical provider/path to an HTTPS remote;
3. derives the bounded cache path from the durable repository ID;
4. prepares the local bare cache if necessary;
5. configures and fetches only the `canonical` remote;
6. resolves the canonical remote HEAD and its default branch name;
7. resolves the exact full object ID of `canonical/HEAD`;
8. reads only the root `README.md` tree entry from that exact tree;
9. distinguishes a missing entry from a present zero-byte blob;
10. rejects tree, submodule, symlink, and other non-regular README modes;
11. checks blob size before materializing content and refuses blobs above the managed-text limit;
12. reads the immutable blob by its tree-entry object ID, preserving exact bytes.

Observation may create or refresh **local cache state** and may perform canonical fetches. It does not create worktree files, commits, tags, branches, or pushes, and it never configures or fetches mirror remotes for README planning. Remote repository state is therefore read-only in this slice.

The branch, base OID, tree entry, mode, and content are all derived after the same canonical fetch. Blob content is read by immutable object ID, so a concurrent remote update cannot change the bytes associated with the emitted base OID; a later planning run fetches and observes the newer canonical state.

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

Changing an existing zero-byte README starts with:

```text
-""
```

The diff finds exact common prefix/suffix line segments, emits at most three unchanged context lines on each side, and JSON-quotes each changed before/after block as one string. Grouping changed blocks bounds JSON-escaping amplification while keeping the existing managed-plan review ceiling sufficient for 1 MiB README inputs.

Review text inherits the managed-text safety policy: invalid UTF-8, terminal controls, Unicode format/bidirectional controls, and Unicode line/paragraph separators fail closed.

## Plan construction

For each configured repository, the builder:

1. validates durable repository and canonical target identity for every configured repository before I/O;
2. sorts configured repositories deterministically;
3. loads the bounded contained template;
4. renders desired README bytes deterministically;
5. requests exact README observation;
6. validates observed branch/base OID/mode/content;
7. omits the repository when a present README already equals desired bytes;
8. otherwise computes the byte-aware review diff;
9. preserves an existing regular-file mode or uses `100644` for creation;
10. records observed raw-content SHA-256, desired SHA-256/content, exact template SHA-256, target identity, and base OID;
11. validates the completed `repora.io/managed-artifact-plan` v1 before returning it.

The plan contains no local template path, cache path, credentials, timestamp, author identity, environment value, or Git command line.

## User-facing review command

`repoctl plan-readme -f repora.yaml` is a separate command from Git-ref `repoctl plan`. This preserves the domain separation required by the managed-artifact architecture and prevents README review from being silently bundled with mirror reconciliation.

The default output is human review text. For each changed repository it prints:

- repository ID and durable UID;
- canonical provider/path/default branch;
- exact reviewed base OID;
- observed and desired README mode/digest state;
- the deterministic byte-aware README review diff.

If no configured README needs a change, the command prints `No managed README changes.` and exits successfully.

`repoctl plan-readme --artifact` emits the exact `repora.io/managed-artifact-plan` v1 JSON instead of human review text. That serialized artifact is evidence/review input only in the current slice: there is no managed README apply command yet.

`plan-readme` accepts only `-f` and `--artifact`. It intentionally does not accept mirror-plan options, `--dry-run`, `--force`, or `--plan-file`; those semantics belong to later exact-plan preflight/apply slices.

## Still deferred

Issue #12 still requires:

- exact-plan stale preflight and dry-run;
- isolated commit creation that changes only root `README.md`;
- guarded canonical push;
- execution result/evidence output;
- fresh mirror reconciliation after a successful canonical README change.
