# Provider/path topology schema v1

## Status

This document describes the preferred `repora.yaml` topology accepted by the current pre-alpha runtime.

Repora separates four concepts:

| Field | Meaning | Stability |
| --- | --- | --- |
| `id` | Human-facing operational alias | May be renamed intentionally |
| `uid` | Durable logical identity used for cache and evidence continuity | Should remain stable |
| `provider + path` | Declarative repository location and mirror selector | Changes when hosting changes |
| resolved URL / Git alias | Runtime transport state | Ephemeral; never durable identity |

## Preferred multi-mirror configuration

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

## Field reference

### `repos`

A non-empty list of repository entries. Entries may be processed concurrently through `--parallel`.

### `id` and `uid`

`id` is the CLI label. `uid` is durable identity. When `uid` is omitted it defaults to `id`, but explicit stable UIDs are recommended.

Both values must be unique within the configuration.

### `canonical`

The authoritative source. The current runtime supports GitLab canonical repositories only.

### `mirrors`

One or more GitHub or GitLab targets.

- Status observes every configured mirror.
- When more than one mirror is configured, every mirror must use provider/path form.
- Duplicate `(provider, path)` targets are rejected.
- Plan/apply/sync remain single-mirror until issue #15 is implemented.

The stable status target is:

```text
<provider>:<path>
```

Array position is order, not identity.

### `path`

A provider-relative repository location without scheme, host, credentials, leading slash, or traversal segments.

Valid examples:

```text
micrantha/anthesis
micrantha/laboratory/dubnium
hackelia-micrantha/repora
```

### `policy.refs`

Optional. Omission normalizes to the closed version-1 policy:

```yaml
version: 1
scope: default-branch-only
destructive: require-force
```

Unsupported versions or broader values fail configuration loading.

### `mode`

Optional. Defaults to `mirror`; no other mode is supported.

## Legacy URL compatibility

A single-mirror entry may continue to use legacy URLs:

```yaml
canonical:
  provider: gitlab
  url: git@gitlab.com:micrantha/anthesis.git
mirrors:
  - provider: github
    url: git@github.com:hackelia-micrantha/anthesis.git
```

Exactly one of `path` or `url` is required per endpoint. Credential-bearing HTTP URLs are rejected.

Legacy URLs are compatibility input, not identity. Multi-mirror entries require provider/path so each target has an unambiguous stable selector. Status v2 derives only a safe repository path from a supported legacy URL and does not expose host, credentials, query values, fragments, or transport details.

## Migration

1. Add an explicit durable `uid` and keep it stable.
2. Replace endpoint URLs with provider-relative paths.
3. Add additional mirrors only after all mirror endpoints use provider/path.
4. Use `repoctl status` to inspect multi-mirror state.
5. Do not expect plan/apply/sync to accept multiple mirrors until the exact multi-mirror execution contract lands.

## Current runtime boundary

Implemented:

- GitLab canonical repositories;
- one or more GitHub/GitLab mirrors for status;
- stable provider/path mirror selectors;
- independent mirror failure reporting;
- default-branch-only closed ref policy;
- exact single-mirror plan artifacts and audited apply;
- runtime HTTPS resolution;
- bounded single-mirror legacy URL input.

Not implemented:

- multi-mirror planning or mutation;
- tags, non-default branches, wildcard refs, or deleted-ref reconciliation;
- provider provisioning;
- custom provider bases or user-selectable transport;
- hosted control-plane behavior.
