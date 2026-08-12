# Provider/path topology schema v1

Status: Current

Repora separates durable identity from runtime transport:

| Field | Meaning | Stability |
| --- | --- | --- |
| `id` | Human-facing operational alias | May be renamed |
| `uid` | Durable logical identity used by cache and evidence | Should remain stable |
| `provider + path` | Declarative canonical or mirror location and selector | Changes only when hosting moves |
| resolved URL / Git alias | Runtime transport state | Ephemeral; never target identity |

## Preferred configuration

```yaml
repos:
  - id: anthesis
    uid: repo.micrantha.anthesis
    canonical:
      provider: gitlab
      path: micrantha/anthesis
    mirrors:
      - provider: github
        path: hackelia-micrantha/anthesis
      - provider: gitlab
        path: micrantha-backup/anthesis
    mode: mirror
    policy:
      refs:
        version: 1
        scope: default-branch-only
        destructive: require-force
```

The runtime derives HTTPS remotes immediately before Git operations. Authentication remains delegated to system Git and credential helpers.

## Fields

### `repos`

A non-empty list. Repository entries may be processed concurrently through `--parallel` after all selected preparation and force authorization succeeds.

### `id` and `uid`

`id` is the CLI label. `uid` is durable identity. An omitted `uid` defaults to `id`, but an explicit stable UID is recommended. Both must be unique.

### `canonical`

The authoritative source. The current runtime supports GitLab canonical repositories only.

### `mirrors`

One or more GitHub or GitLab targets.

- status, exact planning, dry-run, and real mutation support every configured mirror;
- every mirror in a multi-mirror entry must use provider/path form;
- duplicate `(provider, path)` targets are rejected;
- actions are reviewed and executed in configuration order;
- array position and derived Git aliases are not identity;
- imported artifacts bind by provider/path to current runtime aliases.

Stable target form:

```text
<provider>:<path>
```

### `path`

A provider-relative repository location without scheme, host, credentials, leading slash, traversal segments, whitespace, backslashes, or unsafe delimiters.

Examples:

```text
micrantha/anthesis
micrantha/laboratory/dubnium
hackelia-micrantha/repora
```

### `artifacts.readme`

README management is an opt-in configuration extension under issue #12. The configuration/parser and deterministic renderer may land before end-to-end README planning/apply; until the remaining #12 slices are implemented, this block does not change mirror `status`, `plan`, or `apply` behavior.

Shape:

```yaml
artifacts:
  readme:
    template: templates/README.md.tmpl
    values:
      title: Repora
      summary: Deterministic repository mirror management
```

Rules for the configuration slice:

- `readme` is the only recognized artifact field; strict YAML decoding rejects unknown artifact types;
- README management requires canonical provider/path identity; URL-only legacy canonical configuration is rejected for this feature;
- `template` is required and uses a portable slash-separated relative path;
- absolute paths, traversal segments, URLs/drive-style paths, and backslashes are rejected;
- template symlink containment and regular-file checks occur when the template is opened by later planning code;
- `values` is optional inert string data keyed by `[A-Za-z][A-Za-z0-9_-]*` identifiers;
- values are never environment-expanded, shell-evaluated, or recursively interpreted as template syntax;
- configuring README management never grants authority over a path other than repository-root `README.md`.

The dedicated renderer recognizes only `{{repo.id}}`, `{{repo.uid}}`, `{{canonical.provider}}`, `{{canonical.path}}`, and configured `{{value.<key>}}` tokens. Rendering is single-pass, normalizes line endings to LF, rejects malformed/unknown placeholders and invalid UTF-8/NUL text, and caps template and rendered output at 256 KiB.

See [`../architecture/managed-artifacts.md`](../architecture/managed-artifacts.md) and ADR-0017 for the proposed full plan/apply boundary.

### `policy.refs`

Omission normalizes to:

```yaml
version: 1
scope: default-branch-only
destructive: require-force
```

Unsupported versions or broader values fail configuration loading.

### `mode`

Defaults to `mirror`; no other mode is supported.

## Legacy URL compatibility

A single-mirror entry may continue to use legacy URLs. Exactly one of `path` or `url` is required per endpoint, and credential-bearing HTTP URLs are rejected.

Legacy URLs are compatibility transport input, not identity. Multi-mirror entries require provider/path so every target is unambiguous. New exact multi-mirror artifacts and execution evidence never persist transport URLs. README management is stricter and requires canonical provider/path identity even on otherwise-compatible single-mirror entries.

## Execution semantics

Provider/path topology is used consistently across:

- status v2 target identity;
- reconciliation artifact v2 actions;
- current runtime alias rebinding;
- apply v3 source and target results;
- execution-record v3 intent/result evidence.

Before action zero, Repora verifies current configuration, status, policy, default branches, and all expected OIDs. After preflight, mirror actions execute sequentially in exact artifact order. Runtime failure of one mirror does not prevent later independent targets, and successful earlier targets are not rolled back.

## Migration

1. Add and preserve an explicit `uid`.
2. Replace endpoint URLs with provider-relative paths.
3. Add additional mirrors only after every target has unambiguous provider/path identity.
4. Review `repoctl status` and `repoctl plan --artifact`.
5. Run `apply --dry-run` before real mutation.
6. Treat partial apply results as evidence requiring fresh status and planning, not replay or rollback.

## Current runtime boundary

Implemented:

- GitLab canonical repositories;
- one or more GitHub/GitLab mirrors;
- stable provider/path identity through status, planning, execution, results, and journals;
- default-branch-only closed ref policy;
- sequential independent mirror mutation with force-with-lease;
- runtime HTTPS resolution;
- bounded single-mirror legacy URL compatibility;
- strict opt-in `artifacts.readme` configuration parsing and bounded deterministic README rendering primitives.

Not implemented:

- end-to-end managed README observation, plan artifact, dry-run, commit creation, or push;
- tags, non-default branches, wildcard refs, or deleted-ref reconciliation;
- provider provisioning;
- custom provider bases or user-selectable transport;
- concurrent mirror mutation;
- automatic rollback or hosted control-plane behavior.
