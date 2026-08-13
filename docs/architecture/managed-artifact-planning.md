# Managed artifact planning primitives

Status: Current

## Scope

This document describes the implemented managed README lifecycle through guarded canonical mutation: contained local template loading, deterministic rendering, exact canonical Git-tree observation, byte-aware review diff construction, managed-artifact plan assembly, the user-facing `plan-readme` review command, exact-plan stale preflight and dry-run, isolated candidate commit creation in Repora's local bare cache, exact-base leased canonical push, and durable execution journaling/result output.

The planner depends on a `READMEObserver` interface. `NewGitREADMEObserver` provides the production Git-backed implementation. Fresh post-push mirror reconciliation remains owned by issue #12 and is not yet part of this lifecycle.

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

Observation may create or refresh **local cache state** and may perform canonical fetches. It does not create worktree files, commits, tags, branches, or pushes, and it never configures or fetches mirror remotes for README planning.

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

A missing README and a present zero-byte README are also distinct. Creating a zero-byte README includes `+""`; changing an existing zero-byte README starts with `-""`.

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

`repoctl plan-readme -f repora.yaml` is separate from Git-ref `repoctl plan`. This preserves the domain separation required by the managed-artifact architecture and prevents README review from being silently bundled with mirror reconciliation.

The default output is human review text. For each changed repository it prints repository/durable identity, canonical provider/path/default branch, exact reviewed base OID, observed and desired README mode/digest state, and the deterministic byte-aware README review diff.

If no configured README needs a change, the command prints `No managed README changes.` and exits successfully.

`repoctl plan-readme --artifact` emits the exact `repora.io/managed-artifact-plan` v1 JSON instead of human review text.

## Exact-plan stale preflight and dry-run

`PreflightPlan(spec, plan, observer)` validates whether a previously reviewed managed-artifact plan is still safe to execute.

Before observation, preflight:

1. validates the strict managed-artifact plan contract;
2. binds every planned UID to current configuration;
3. requires managed README authority to still be explicitly enabled;
4. re-validates current durable repository identity and canonical provider/path;
5. requires current repository ID and canonical provider/path to match the reviewed plan.

Only after all planned configuration bindings pass does preflight observe repositories. For each repository it then requires current canonical default branch, exact `base_oid`, README presence, regular-file mode/content digest, and recomputed review diff to match the reviewed plan.

Configuration or repository-state mismatches return `ErrStale`. Transport/cache/observation failures are operational errors rather than stale-plan results.

`repoctl apply-readme -f repora.yaml --plan-file FILE --dry-run` exposes read-only preflight. A stale plan exits with status 2; invalid plans or operational failures exit with status 1. A successful dry-run prints the same human review representation as `plan-readme` and creates no commit or push.

Exact-plan execution does not re-render the current template. The reviewed desired content in the plan is self-contained execution authority. Removing README authority or changing durable/canonical identity still invalidates the plan.

## Isolated local commit preparation

`CommitPreparer.Prepare(spec, plan, observer)` always runs exact stale preflight first. Only after preflight succeeds does it create otherwise-unreferenced objects in Repora's existing bare cache.

For each planned repository it writes the reviewed desired README blob, rebuilds the reviewed base root tree with only root `README.md` replaced/added, creates one child commit of the exact reviewed `base_oid`, and verifies recursively that only `README.md` changed and that mode/content/digest match reviewed desired state.

The commit message is fixed to `chore: update managed README`; author and committer are fixed to `Repora <repora@localhost.invalid>` with one current UTC execution instant.

Candidate creation writes Git objects only. It does not update a branch, tag, remote-tracking ref, local HEAD, worktree, or remote repository. Candidate OIDs are execution evidence rather than deterministic plan fields because commit metadata includes execution time.

## Guarded canonical push

`Pusher.Push(spec, plan, prepared, observer)` owns the managed README remote-mutation boundary. Its Git dependency can only re-read candidate objects, resolve revisions, and push a branch with an exact lease.

Before the first push it:

1. validates the strict plan and one prepared candidate per planned repository;
2. requires candidate UID/ID/base bindings to match the plan;
3. requires each candidate parent to equal the reviewed `base_oid`;
4. requires the candidate tree OID to match its prepared evidence;
5. re-verifies recursively that the candidate changes only `README.md` and matches reviewed mode/content/digest;
6. runs one fresh `PreflightPlan` across all planned repositories.

Each sequential remote mutation then uses:

```text
--force-with-lease=refs/heads/<reviewed-branch>:<reviewed-base-oid>
<candidate-oid>:refs/heads/<reviewed-branch>
```

The candidate is already verified as a direct child of the reviewed base, so the expected unchanged transition is a fast-forward. The exact lease closes the race between fresh preflight and push and refuses to overwrite/recreate a branch whose current remote OID is no longer the reviewed base.

Multi-repository remote mutation is not atomic. `PushResult` preserves successful earlier pushes plus a failed later attempt so partial success is never represented as all-or-nothing.

## Journaled real apply

`repoctl apply-readme -f repora.yaml --plan-file FILE` now exposes real exact-plan execution. It does not accept a force override.

The execution order is fixed:

1. load and strictly validate the reviewed managed-artifact plan;
2. initialize the protected journal root beside the configuration file;
3. persist `repora.io/managed-artifact-execution-record` v1 `INTENT` **before candidate-object creation or remote mutation**;
4. prepare locally verified candidate commits (including their own stale preflight);
5. run the guarded pusher (including fresh full preflight and exact per-branch leases);
6. construct a result that preserves prepared commit IDs and per-repository pushed/outcome state;
7. persist the matching journal `RESULT` for success, stale, preparation failure, push failure, or partial multi-repository success;
8. print human result output or, with `--json`, `repora.io/managed-artifact-apply-result` v1.

If INTENT persistence fails, candidate preparation and push are not entered. If RESULT persistence fails after remote mutation, the CLI returns an error but still emits the projected apply result so successful mutation is not hidden.

Journal records bind to SHA-256 of the canonical serialized managed plan plus repository target/base/desired mode/digest. They intentionally do **not** duplicate desired README content, local paths, credentials, timestamps from the plan, or raw operational error text. Failure is represented by bounded phase/outcome metadata; redacted operational diagnostics remain on stderr.

The managed-artifact execution record is separate from the Git-ref `PUSH_BRANCH` execution-record schema, while both use the same protected `.repora/journal` no-overwrite/fsync persistence mechanism.

## Still deferred

Issue #12 still requires:

- a fresh post-push mirror reconciliation step after successful canonical README mutation.
