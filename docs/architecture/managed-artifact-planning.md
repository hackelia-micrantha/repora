# Managed artifact planning and apply

Status: Current

## Scope

This document describes the implemented managed README lifecycle: contained local template loading, deterministic rendering, exact canonical Git-tree observation, byte-aware review diff construction, managed-artifact plan assembly, user-facing review, exact-plan stale preflight/dry-run, isolated candidate commit creation, exact-base leased canonical push, and durable execution evidence.

Mirror propagation is intentionally **not** part of managed-artifact apply. A successful canonical README mutation invalidates prior mirror observations/plans; propagation uses a separate fresh `status → plan → apply` review cycle.

## Input boundary

`BuildPlan(configPath, spec, observer)` processes only repositories with explicit `artifacts.readme` configuration.

- No configured README artifact produces an explicit empty plan without template or observer I/O.
- Repository IDs, durable UIDs, and canonical provider/path identities are validated before artifact I/O.
- Duplicate managed UIDs, IDs, or canonical targets fail before I/O.
- Managed repositories require provider/path canonical identity rather than legacy URL-only topology.
- Configured repositories are ordered deterministically by durable UID then ID.

The normal configuration loader owns broader topology/ref-policy validation. Managed-artifact planning repeats the identity checks required by its own durable authority boundary.

## Template loading and rendering

`LoadTemplate` resolves the physical configuration file and treats its containing directory as the template root. The configuration and resolved template must be regular files.

Template references must be canonical portable relative paths and may not use absolute paths, URLs, `~`, backslashes, traversal, redundant segments, unsafe controls, or symlink escapes outside the configuration root. Template input is bounded to 1 MiB.

The renderer supports only:

```text
{{repo.id}}
{{repo.uid}}
{{canonical.provider}}
{{canonical.path}}
{{value.<key>}}
```

Rendering is deterministic and single-pass. Replacement values are inert text and are not recursively interpreted. Invalid UTF-8, NUL, unsupported display/control characters, malformed/unresolved tokens, and oversized output fail closed. Line endings are normalized to LF.

The plan records SHA-256 of the exact loaded template bytes, never the local template path.

A complete validated example is maintained under [`../../examples/managed-readme/`](../../examples/managed-readme/).

## Canonical README observation

The production `READMEObserver` uses Repora's existing bare-cache/HTTPS transport boundary. After a fresh canonical fetch it observes:

```text
READMEObservation {
  branch
  base_oid
  present
  mode
  raw content bytes
}
```

The observer:

1. validates durable/canonical identity;
2. configures/fetches only the canonical remote;
3. resolves the canonical default branch and exact full HEAD OID;
4. reads root `README.md` from that exact tree;
5. distinguishes missing from present-empty;
6. accepts only regular blob modes `100644`/`100755`;
7. checks blob size before materializing content;
8. preserves exact observed bytes for digest/review comparison.

Observation may create or refresh local Repora cache state. It does not write a user checkout, create commits/tags/branches, configure mirrors, or push.

## Byte-aware review diff

`ReviewDiff` is deterministic review evidence, not an applyable patch. It uses fixed `a/README.md` / `b/README.md` labels and JSON-quotes exact changed byte-backed strings so control sequences cannot alter terminal structure.

Line-ending-only changes remain visible, and missing README is distinct from a present zero-byte README. Review text is bounded and subject to managed-text safety validation.

## Exact plan construction

For each configured repository, planning:

1. validates durable and canonical identity;
2. loads and renders the contained template;
3. obtains exact canonical README observation;
4. validates branch/base/mode/content state;
5. omits the repository when current bytes already equal desired bytes;
6. otherwise computes deterministic review diff;
7. preserves existing regular-file mode or uses `100644` for creation;
8. records observed digest, exact desired content/digest, template digest, target identity, and base OID;
9. validates the completed `repora.io/managed-artifact-plan` v1 contract.

The plan contains no local path, credentials, timestamp, environment value, author identity, resolved transport URL, or Git command line.

## Review command

```bash
repoctl plan-readme -f repora.yaml
repoctl plan-readme -f repora.yaml --artifact > readme-plan.json
```

Human output includes repository/durable identity, canonical target/default branch, exact reviewed base OID, observed/desired README state, and deterministic review diff.

An empty plan prints `No managed README changes.` and succeeds. `--artifact` emits the exact validated v1 JSON plan.

## Exact stale preflight and dry-run

`PreflightPlan` first validates the strict plan and binds every planned UID back to current configuration. It requires managed README authority to remain enabled and current repository ID/canonical provider/path to match the reviewed plan.

Only after configuration binding does it re-observe canonical state. Current branch, exact base OID, README presence/mode/content digest, and recomputed review diff must all equal the reviewed plan. State/config mismatches return typed stale state; transport/cache/observation failures remain operational errors.

```bash
repoctl apply-readme -f repora.yaml --plan-file readme-plan.json --dry-run
```

Dry-run performs the same exact current-state validation and prints the reviewed diff without creating commits or pushing. A stale managed plan exits `2`; invalid input or operational failure exits `1`.

Apply does not re-render the current template. The desired content already reviewed in the exact plan is execution authority; removal of README authority or identity drift invalidates that plan.

## Isolated candidate commit preparation

After exact preflight, `CommitPreparer` creates otherwise-unreferenced objects in Repora's bare cache:

- writes the exact reviewed README blob;
- reconstructs the reviewed base tree with only root `README.md` replaced/added;
- creates one direct child commit of the reviewed base;
- verifies recursively that exactly `README.md` changed;
- re-verifies desired mode/content/digest.

Candidate creation does not update any branch/tag/HEAD/ref, worktree, or remote. Candidate OIDs are execution evidence rather than deterministic plan fields because execution commit metadata includes the current execution instant.

## Guarded canonical push

Before mutation the pusher validates plan/candidate cardinality and exact UID/ID/base/tree/content bindings, re-reads candidate objects, verifies the candidate is a direct child of the reviewed base, and runs one fresh full preflight across all planned repositories.

Each sequential push uses the reviewed branch and exact reviewed base as the lease:

```text
--force-with-lease=refs/heads/<branch>:<base_oid>
<candidate_oid>:refs/heads/<branch>
```

The candidate itself is a verified child of the reviewed base, so the expected unchanged transition is a fast-forward. The explicit lease closes the race between fresh preflight and remote mutation.

Managed README apply has no `--force` override.

## Journaled real apply

```bash
repoctl apply-readme -f repora.yaml --plan-file readme-plan.json
repoctl apply-readme -f repora.yaml --plan-file readme-plan.json --json
```

The fixed ordering is:

1. load and strictly validate exact plan;
2. initialize protected journal root;
3. persist `repora.io/managed-artifact-execution-record` v1 `INTENT` before candidate creation or push;
4. prepare locally verified candidate commits;
5. perform fresh preflight and exact leased canonical pushes;
6. build per-repository result preserving prepared commit IDs, pushed state, and outcome;
7. persist matching `RESULT` for success, stale, preparation failure, push failure, or partial success;
8. print human output or `repora.io/managed-artifact-apply-result` v1 JSON.

INTENT persistence failure prevents candidate preparation/mutation. If RESULT persistence fails after remote mutation, the command fails but still emits the projected apply result so successful mutation is not hidden.

Journal evidence binds canonical serialized plan digest plus target/base/desired mode/digest. It does not duplicate desired README bodies, credentials, raw command lines, or unbounded operational errors.

## Partial success and non-atomicity

Multi-repository managed README mutation is intentionally non-atomic. A result can preserve earlier `APPLIED` repositories and a later `FAILED` repository. Earlier successful pushes are never automatically rolled back.

Recovery requires fresh canonical observation and a new exact plan. Old plans and journal records are evidence, never replay authorization.

## Fresh mirror reconciliation

A successful managed README push changes canonical HEAD, so any mirror status/plan observed before that change is stale by construction.

Required operator sequence:

```text
managed README plan
  -> review
  -> managed README apply
  -> fresh mirror status
  -> fresh mirror plan
  -> mirror apply
```

This is the completed architecture, not an unimplemented #12 step. Repora deliberately does not bundle managed artifact mutation with mirror mutation and does not reuse an earlier mirror plan.

## Current boundaries

Still intentionally unsupported:

- arbitrary managed file paths or generic file generation;
- remote/executable templates;
- force override for managed artifact apply;
- automatic mirror propagation;
- cross-repository transactions or automatic rollback;
- provider API mutation;
- Anthesis runtime policy coupling.
