# Managed artifact architecture

Status: Current

ADR-0017 defines the managed-artifact boundary and issue #12 delivered root `README.md` as the first concrete artifact type. This document describes the current implemented authority model; detailed planner/executor behavior is in [`managed-artifact-planning.md`](managed-artifact-planning.md).

## Purpose

Managed artifacts let Repora propose and apply narrowly scoped repository-content changes without becoming a general-purpose file generator.

V1 intentionally supports one artifact:

- type: README;
- managed path: repository-root `README.md`;
- mutation target: canonical default branch;
- template source: local file under the configuration root;
- renderer: deterministic token replacement only.

The Git-ref reconciliation plan remains unchanged and Git-ref-only.

## Ownership model

README management is opt-in per repository. Without an `artifacts.readme` block, managed-artifact planning and apply have no effect on that repository.

Once configured, Repora may propose replacement of root `README.md` content only. Configuration grants no authority over any other repository path. Existing README bytes and regular-file Git mode are observed state and cannot change without an exact reviewed managed-artifact plan.

If `README.md` resolves as a symlink, submodule, tree, or another non-regular entry, planning fails closed. Existing regular blob modes `100644` and `100755` are preserved; a newly created README uses `100644`.

Future artifact types do not inherit README authority. Each requires its own reviewed contract and implementation boundary.

## Configuration

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
          summary: Deterministic repository control
```

A runnable shape is maintained under [`../../examples/managed-readme/`](../../examples/managed-readme/).

### Template path rules

`template` is resolved relative to the physical directory containing `repora.yaml`. Planning rejects absolute or non-canonical paths, traversal, escapes through symlinks, remote URLs, non-regular files, unsafe display/control characters, and templates above the v1 size limit.

The resolved local path is runtime input and never appears in the durable plan. The plan records the exact template-content digest instead.

### Values and renderer

`values` is an optional string-to-string map. Supported exact tokens are:

```text
{{repo.id}}
{{repo.uid}}
{{canonical.provider}}
{{canonical.path}}
{{value.<key>}}
```

Rendering is one deterministic pass. Values are inert and are not recursively interpreted as template syntax. The renderer normalizes line endings to LF and rejects malformed/unknown tokens, invalid UTF-8, NUL, unsafe display/control characters, oversized templates/output, and unresolved referenced values.

There are no functions, loops, conditions, includes, shell commands, plugins, environment/secret interpolation, remote templates, or network requests.

## Plan contract

Managed-artifact plans are separate from mirror-reconciliation artifacts. The implemented contract is `repora.io/managed-artifact-plan` v1.

Each planned repository binds:

- durable repository `uid` and human `id`;
- canonical provider/path/default branch;
- exact canonical `base_oid`;
- action type `WRITE_README` and fixed path `README.md`;
- observed README presence, regular-file mode, and content SHA-256;
- desired regular-file mode, exact UTF-8 content, and SHA-256;
- exact template SHA-256;
- deterministic byte-aware review diff.

The plan excludes timestamps, execution identity, credentials, resolved remote URLs, local template/cache paths, environment values, and Git command lines.

The versioned schema under `schemas/` is authoritative for serialized shape.

## Planning flow

```text
repora.yaml
  -> strict artifact config/identity validation
  -> load contained local template
  -> deterministic render
  -> fresh canonical default-branch observation
  -> inspect exact README tree entry/mode/blob
  -> equal bytes? omit action
  -> different? exact managed-artifact plan + review diff
```

`repoctl plan-readme -f repora.yaml` renders human review output. `--artifact` emits the exact v1 plan.

Planning may refresh Repora's local canonical cache, but does not create commits, update refs, write a user checkout, or push.

## Dry-run and stale preflight

`repoctl apply-readme -f repora.yaml --plan-file FILE --dry-run` consumes the same exact reviewed plan as real apply.

Before execution authority is accepted, Repora rebinds the durable repository to current configuration and re-observes canonical provider/path/branch, exact base OID, README presence/mode/content digest, and the deterministic review diff. Any mismatch is stale and fails before mutation.

The reviewed desired bytes in the plan are execution authority; apply does not silently re-render a changed template.

## Real apply flow

A real apply uses the fixed sequence:

```text
strict plan validation
  -> protected managed-artifact INTENT persistence
  -> exact stale preflight
  -> isolated candidate commit creation
  -> candidate parent/tree/path/mode/content verification
  -> fresh full preflight
  -> exact reviewed-base leased canonical push
  -> managed-artifact RESULT persistence
```

The candidate commit is a direct child of the reviewed base and changes exactly root `README.md`. Candidate creation uses Repora's local bare cache and does not mutate an arbitrary developer checkout or local ref.

Each remote mutation is guarded by the reviewed branch and exact base lease. Managed README apply has no `--force` option.

The public apply-result and execution-evidence contracts are `repora.io/managed-artifact-apply-result` v1 and `repora.io/managed-artifact-execution-record` v1.

## Multi-repository results and recovery

Managed README mutation across repositories is not transactional. An earlier canonical push may succeed before a later repository fails. Results and journal evidence preserve that state rather than representing false atomicity.

INTENT persistence failure prevents candidate creation and mutation. If RESULT persistence fails after a successful remote push, the command still fails but emits the projected apply outcome so mutation is not hidden.

Recovery requires observing current canonical state and creating a fresh plan. Repora never replays a previous managed-artifact plan as authorization and never performs automatic rollback.

## Mirror interaction

Managed artifact mutation and mirror reconciliation are separate domains and separate review cycles.

After a managed README apply changes canonical HEAD, all mirror observations/plans made against the previous canonical state are stale. Propagation therefore uses a fresh operator cycle:

```text
managed artifact plan
  -> review
  -> managed artifact apply
  -> fresh status
  -> fresh mirror plan
  -> mirror apply
```

This fresh mirror cycle is not a missing managed-artifact implementation step. Repora deliberately does **not** auto-apply mirrors or reuse a pre-README mirror plan.

## Safety invariants

V1 preserves these boundaries:

- no artifact config => no artifact action;
- one managed output path only: `README.md`;
- no configurable output path or generic file writer;
- existing regular mode preserved; missing README becomes `100644`;
- symlink/submodule/tree/non-text/oversized unsafe state fails closed;
- no executable or remote template behavior;
- no template/cache paths, credentials, or environment values in durable plans;
- no real apply without exact current-state validation;
- no user-checkout mutation;
- no force override;
- no automatic mirror apply after README mutation;
- no cross-repository atomicity or automatic rollback;
- no future artifact type without a separately reviewed authority boundary.
