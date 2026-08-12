# Managed artifact architecture

Status: Proposed

Issue #7 defines the managed-artifact boundary; issue #12 is the README implementation slice. Nothing in this document is current runtime behavior until the corresponding source and tests are merged.

## Purpose

Managed artifacts let Repora propose and apply narrowly scoped repository-content changes without becoming a general-purpose file generator.

V1 is intentionally one artifact:

- type: README;
- managed path: repository-root `README.md`;
- mutation target: canonical default branch;
- template source: local file under the configuration root;
- renderer: deterministic token replacement only.

The Git-ref reconciliation plan remains unchanged and Git-ref-only.

## Ownership model

README management is opt-in per repository. Without an `artifacts.readme` configuration block, artifact planning and apply have no effect on that repository.

Once configured, Repora may propose replacement of root `README.md` content only. Configuration does not grant authority over any other repository path. Existing README content and regular-file Git mode are treated as observed state and are never changed without an explicit reviewed plan.

If `README.md` currently resolves in the Git tree as a symlink, submodule, tree, or other non-regular entry, planning fails instead of silently converting it into a regular text file. Existing regular blob modes `100644` and `100755` are preserved; a newly created README uses `100644`.

Future artifact types do not inherit README authority. Each one needs separate design approval.

## Proposed configuration

```yaml
repos:
  - id: repora
    uid: repo.repora
    canonical:
      provider: gitlab
      path: micrantha/repora
    mirrors:
      - provider: github
        path: hackelia-micrantha/repora
    artifacts:
      readme:
        template: templates/README.md.tmpl
        values:
          title: Repora
          summary: Deterministic repository mirror management
```

### Template path rules

`template` is resolved relative to the directory containing `repora.yaml`.

Planning must reject:

- absolute paths;
- `..` traversal outside the configuration root;
- paths that resolve outside the configuration root through symlinks;
- non-regular template files;
- remote URLs;
- templates above the v1 size limit.

The resolved local path is runtime input and must not appear in a durable plan artifact. The plan records the template content digest instead.

### Values

`values` is an optional string-to-string map. Keys are symbolic identifiers suitable for `{{value.<key>}}` placeholders. Values are inert data; inserted values are never recursively interpreted as template syntax.

No environment-variable or secret interpolation is supported.

## Renderer v1

Supported exact tokens are:

```text
{{repo.id}}
{{repo.uid}}
{{canonical.provider}}
{{canonical.path}}
{{value.<key>}}
```

Rules:

- render in one deterministic pass;
- normalize text line endings to LF;
- reject invalid UTF-8 or NUL bytes in templates, values, and observed README text;
- reject unknown or malformed placeholders;
- reject missing configured values referenced by the template;
- do not process placeholder-looking text introduced by a replacement value;
- do not execute functions, loops, conditions, includes, commands, plugins, or network requests.

The implementation should use a small dedicated renderer rather than a general template engine. Template and rendered output sizes must have a fixed v1 upper bound appropriate for README text.

## Plan contract

Managed artifact plans are separate from reconciliation artifacts.

Proposed v1 JSON shape:

```json
{
  "kind": "repora.io/managed-artifact-plan",
  "version": 1,
  "repositories": [
    {
      "uid": "repo.repora",
      "id": "repora",
      "target": {
        "provider": "gitlab",
        "path": "micrantha/repora",
        "branch": "main"
      },
      "base_oid": "<canonical-head>",
      "actions": [
        {
          "type": "WRITE_README",
          "path": "README.md",
          "observed": {
            "present": true,
            "mode": "100644",
            "sha256": "<digest>"
          },
          "desired": {
            "mode": "100644",
            "sha256": "<digest>",
            "content": "# Repora\n...\n"
          },
          "template_sha256": "<digest>",
          "diff": "--- a/README.md\n+++ b/README.md\n..."
        }
      ]
    }
  ]
}
```

For a missing README, `observed.present` is false and observed mode/digest use the schema's explicit absent representation. Desired mode is `100644`. For an existing regular README, desired mode must equal observed mode so README management cannot change the executable bit implicitly.

The exact schema belongs to #12, but the following boundaries are fixed by ADR-0017:

- artifact kind/version are explicit;
- repository `uid` remains durable identity;
- target identity uses canonical provider/path/branch, not a resolved URL;
- output path is exactly `README.md`;
- plan includes exact `base_oid`;
- observed README state has presence, Git mode, and content digest;
- desired state has reviewed Git mode, digest, and exact UTF-8 content;
- template evidence uses a digest, never its local path;
- review output includes deterministic text diff;
- no timestamp or execution identity appears in deterministic planning.

## Deterministic diff

The human diff must be reproducible from the same observed and desired bytes.

V1 requirements:

- fixed path labels (`a/README.md`, `b/README.md`);
- no timestamps or temporary paths;
- LF line endings;
- stable hunk ordering;
- missing README represented consistently as file creation;
- no terminal color/control sequences in serialized diff.

The text diff covers content. Git mode is separately explicit in observed/desired structured state. The plan validator must reject a desired-content digest mismatch or an unsupported desired mode. Apply recomputes the diff from current observed bytes and planned desired content after stale preflight and requires it to match the reviewed diff.

## Read-only planning flow

```text
repora.yaml
  -> strict artifact config validation
  -> resolve canonical provider/path
  -> observe canonical default branch + exact OID
  -> inspect README tree entry + mode + blob at that exact tree
  -> require absent or regular UTF-8 text blob
  -> load bounded local template
  -> deterministic render
  -> equal bytes? no action
  -> different? build managed-artifact plan + review diff
```

Planning must not create commits, modify caches in a way that changes remote intent, write a user checkout, or push.

## Apply flow

A real artifact apply consumes an exact reviewed managed-artifact plan.

Before any remote push:

1. validate plan kind/version and strict fields;
2. bind UID to current configuration;
3. require current canonical provider/path/default branch to match;
4. re-observe canonical HEAD and require exact `base_oid`;
5. re-read root `README.md` and require exact observed presence, regular-file mode, and content digest;
6. validate desired mode, content, and digest;
7. recompute and verify the reviewed text diff;
8. prepare a commit that changes only root `README.md` content and preserves the reviewed mode;
9. publish with an exact expected-head guard.

Any mismatch is stale and fails before remote mutation.

### Git workspace boundary

Apply must not edit an arbitrary local checkout. Implementation should use isolated Git plumbing or an isolated temporary workspace tied to Repora's runtime cache. Temporary paths remain runtime details and are excluded from durable plan identity.

The resulting commit is a direct child of the reviewed `base_oid`. Commit author/committer metadata and timestamp are execution details; deterministic planning does not predict the resulting commit OID.

## Dry-run

Dry-run consumes the same exact plan and performs the same validation/stale preflight through the point immediately before commit creation/push. It reports the same reviewed content diff and performs no remote mutation.

## Mirror interaction

Content mutation and mirror reconciliation are separate domains and separate review cycles.

After artifact apply succeeds, canonical HEAD changes. Any earlier mirror reconciliation artifact is stale and must not be reused.

Required order:

```text
managed artifact plan
  -> review
  -> managed artifact apply
  -> fresh status
  -> fresh mirror plan
  -> mirror apply
```

Repora must not silently bundle the README commit with a previously observed mirror plan.

## Result and recovery

Artifact apply results should identify:

- repository UID/ID;
- canonical provider/path/branch;
- reviewed base OID;
- resulting commit OID when applied;
- observed and desired README Git mode and content digests;
- outcome (`APPLIED`, `STALE`, `SKIPPED`, or `FAILED`);
- sanitized error.

There is no automatic rollback. A failed or stale operation is recovered by inspecting current canonical state and creating a new plan. A successful unwanted README commit can be reverted through normal Git history; Repora does not rewrite history automatically.

## Safety invariants

V1 must preserve all of these:

- no artifact config => no artifact action;
- one managed path only: `README.md`;
- no output-path configuration;
- absent README becomes regular `100644`; existing regular README mode is preserved;
- symlink/submodule/tree/binary README state fails planning rather than being replaced;
- no executable template behavior;
- no remote templates;
- no implicit GitHub/GitLab API mutations;
- no template/cache paths serialized in plans;
- no credentials or environment values serialized in plans;
- no apply without exact current-state validation;
- no mutation of a user checkout;
- no automatic mirror apply after README mutation;
- no future artifact type without a separate issue or ADR.

## Implementation slices for #12

A simple implementation order is:

1. config shape + validation;
2. bounded renderer + golden tests;
3. domain-specific managed-artifact plan schema/parser, including mode/content preconditions;
4. read-only canonical README observation and deterministic diff planning;
5. exact-plan dry-run/stale validation;
6. isolated commit creation + guarded canonical push;
7. result rendering and evidence;
8. end-to-end tests proving unconfigured repositories are untouched, non-regular README state is rejected, and stale plans fail closed.

Each slice should remain reviewable independently rather than introducing a generic artifact framework first.
