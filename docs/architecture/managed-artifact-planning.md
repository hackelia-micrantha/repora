# Managed artifact planning primitives

Status: Current

## Scope

This document describes the implemented managed README planning and local preparation layers: contained local template loading, deterministic rendering, exact canonical Git-tree observation, byte-aware review diff construction, managed-artifact plan assembly, the user-facing `plan-readme` review command, exact-plan dry-run stale preflight, and isolated candidate commit creation in Repora's local bare cache.

`NewGitREADMEObserver` provides the production Git-backed observation implementation. Guarded remote push and post-push reconciliation are **not** implemented by this layer and remain owned by issue #12.

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

A template reference must be relative and canonical, slash-separated and portable, and free of absolute paths, URLs, `~`, backslashes, traversal, redundant segments, unsafe controls, and surrounding whitespace.

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

1. re-validates durable ID and provider/path canonical identity;
2. resolves the canonical provider/path to HTTPS;
3. prepares Repora's local bare cache;
4. configures and fetches only the `canonical` remote;
5. resolves the canonical default branch and exact `canonical/HEAD` object ID;
6. reads only root `README.md` from that exact tree;
7. distinguishes missing from present-empty;
8. rejects non-regular modes;
9. bounds blob size before materializing content;
10. reads exact immutable blob bytes.

Observation may create or refresh local cache state and may perform canonical fetches. It does not create worktree files, commits, tags, branches, or pushes, and it never configures or fetches mirror remotes for README planning.

## Byte-aware review diff

`ReviewDiff` is a deterministic review representation, not an applyable patch. It preserves exact byte-visible line endings through JSON-quoted content, distinguishes missing and present-empty README state, emits bounded context, and rejects unsafe display controls.

## Plan construction

For each configured repository, the builder validates durable identity, loads and renders the bounded template, observes exact canonical README state, omits no-op repositories, computes the byte-aware review diff, preserves an existing regular-file mode or uses `100644` for creation, records exact observed/desired/template digests and base OID, and validates the completed `repora.io/managed-artifact-plan` v1.

The plan contains no local template path, cache path, credentials, timestamp, author identity, environment value, or Git command line.

## User-facing review command

`repoctl plan-readme -f repora.yaml` is separate from Git-ref `repoctl plan`. Default output is human review text containing repository/durable identity, canonical target/default branch, exact base OID, observed/desired README mode and digest state, and the deterministic review diff.

`repoctl plan-readme --artifact` emits the exact `repora.io/managed-artifact-plan` v1 JSON.

## Exact-plan dry-run preflight

`PreflightPlan(spec, plan, observer)` validates whether a previously reviewed managed-artifact plan is still safe to execute.

Before observation it validates the strict plan, binds every planned UID to current configuration, requires README authority to remain enabled, re-validates durable/canonical identity, and requires current repository ID and provider/path to match the reviewed plan.

After those bindings pass, it requires current default branch, exact canonical HEAD, README presence, regular-file mode, content digest, and recomputed review diff to match the reviewed plan. Configuration or repository-state mismatches return `ErrStale`; transport/cache failures remain operational errors.

`repoctl apply-readme -f repora.yaml --plan-file FILE --dry-run` exposes this preflight. `--dry-run` remains mandatory in the current CLI; no current command pushes a managed README change.

## Isolated local commit preparation

`CommitPreparer.Prepare(spec, plan, observer)` adds the first local mutation boundary. It always runs exact stale preflight first. Only after preflight succeeds does it create otherwise-unreferenced objects in Repora's existing bare cache.

For each planned repository it:

1. writes the reviewed desired README content as a local Git blob;
2. reads the reviewed base commit's root tree;
3. replaces or adds only the root `README.md` entry with the reviewed `100644` or `100755` mode;
4. creates a new tree object without using or mutating a shared Git index;
5. creates one child commit whose parent is exactly the reviewed `base_oid`;
6. uses fixed local execution identity `Repora <repora@localhost.invalid>` and one current UTC instant for author/committer timestamps;
7. verifies recursively that the candidate commit changes exactly one path, `README.md`;
8. re-reads the candidate `README.md` and requires its mode, exact bytes, and SHA-256 to equal reviewed desired state.

The commit message is the fixed Conventional Commit message `chore: update managed README`.

Candidate creation writes Git objects only. It does **not** update a branch, tag, remote-tracking ref, local HEAD, worktree, or remote repository. A failed multi-repository preparation may therefore leave unreachable local objects in the cache; these objects confer no execution authority and are eligible for normal Git object cleanup.

The commit preparer's Git dependency intentionally contains no ref-update or push capability. The returned `PreparedCommit` values contain only UID/ID, reviewed base OID, candidate tree OID, and candidate commit OID for a later guarded-push slice.

Commit OIDs are execution evidence rather than deterministic plan fields because Git commit metadata includes execution time.

## Still deferred

Issue #12 still requires:

- guarded canonical push with an exact reviewed-base lease;
- execution result/evidence output;
- fresh mirror reconciliation after a successful canonical README change.
