# Provider/Path Topology Schema v1

## Status

This document describes the preferred `repora.yaml` topology accepted by the current pre-alpha runtime.

Repora identifies a configured repository through four distinct concepts:

| Field | Meaning | Stability |
| --- | --- | --- |
| `id` | Human-facing operational alias used in CLI output and errors | May be renamed intentionally |
| `uid` | Durable logical identity used for cache and future history/evidence continuity | Should remain stable |
| `provider + path` | Authoritative provider-relative repository location | Changes when repository hosting changes |
| resolved URL | Runtime Git transport address derived immediately before Git operations | Ephemeral runtime state |

Do not use a transport URL as durable repository identity.

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
    mode: mirror
```

The current runtime derives HTTPS remotes:

```text
https://gitlab.com/micrantha/anthesis.git
https://github.com/hackelia-micrantha/anthesis.git
```

The resolver also supports SSH construction internally, but user-configurable transport selection is not implemented. Status and apply currently use the default HTTPS resolver for path-based endpoints.

## Field reference

### `repos`

A non-empty list of repository synchronization entries. Multiple entries may be processed in one invocation.

### `id`

Required human-facing alias. It appears in CLI and JSON results and must be unique within the file.

Changing `id` changes the operational label but should not move durable cache or future journal identity when `uid` remains unchanged.

### `uid`

Optional in the current compatibility model. When omitted, Repora defaults it to `id`.

For durable operation, set an explicit `uid` and do not change it when:

- renaming `id`
- moving a repository between groups or organizations
- changing providers
- changing transport

`uid` values must be unique within the file.

### `canonical`

The authoritative source repository. The current runtime supports GitLab canonical repositories only.

Preferred form:

```yaml
canonical:
  provider: gitlab
  path: group/subgroup/repository
```

GitLab nested namespaces are supported.

### `mirrors`

Target repositories that should converge to canonical state. The schema is a list for forward compatibility, but the current runtime requires exactly one mirror per repository entry.

The mirror provider may currently be GitHub or GitLab.

### `provider`

Selects the provider-specific transport base and validation behavior. Current built-in providers are:

- `gitlab`
- `github`

Custom provider bases are not yet configurable.

### `path`

Provider-relative repository location without scheme, host, credentials, or leading slash.

Valid examples:

```text
micrantha/anthesis
micrantha/laboratory/dubnium
hackelia-micrantha/repora
```

Invalid examples:

```text
https://gitlab.com/micrantha/anthesis.git
git@gitlab.com:micrantha/anthesis.git
/micrantha/anthesis
../anthesis
```

Repora adds the `.git` suffix during runtime resolution when needed.

### `mode`

Optional. Defaults to `mirror`. No other mode is currently supported.

## URL derivation and credentials

Resolved URLs are transport details and must not be written into plans, journals, receipts, identity keys, or other durable topology artifacts.

Credentials must not be stored in `repora.yaml`. Repora delegates authentication to system Git, SSH configuration, and Git credential helpers.

Credential-bearing HTTP URLs are rejected during configuration loading:

```yaml
# Rejected
url: https://token@github.com/org/repository.git
```

SCP-style SSH URLs remain accepted by the legacy compatibility path because `git@host:path` identifies an SSH user, not an embedded secret.

## Legacy URL compatibility

Existing URL-based entries remain accepted temporarily:

```yaml
repos:
  - id: anthesis
    uid: repo.micrantha.anthesis
    canonical:
      provider: gitlab
      url: git@gitlab.com:micrantha/anthesis.git
    mirrors:
      - provider: github
        url: git@github.com:hackelia-micrantha/anthesis.git
```

For each endpoint, exactly one of `path` or legacy `url` must be present. New configuration should use `path`.

Legacy URLs are compatibility input, not authoritative identity. They will be retired after transport configuration and migration behavior stabilize.

## Migration

### Add durable identity

Before:

```yaml
- id: anthesis
  canonical:
    provider: gitlab
    url: git@gitlab.com:micrantha/anthesis.git
```

After:

```yaml
- id: anthesis
  uid: repo.micrantha.anthesis
  canonical:
    provider: gitlab
    url: git@gitlab.com:micrantha/anthesis.git
```

Adding `uid` is non-destructive. Choose it once and keep it stable.

### Replace URLs with provider paths

Before:

```yaml
canonical:
  provider: gitlab
  url: git@gitlab.com:micrantha/laboratory/dubnium.git
```

After:

```yaml
canonical:
  provider: gitlab
  path: micrantha/laboratory/dubnium
```

Do not retain both fields on the same endpoint.

Changing `provider` or `path` while retaining `uid` changes where Repora observes and synchronizes the repository without changing its durable logical identity.

## Actionable validation failures

Representative failures include:

```text
canonical provider is required for repo "anthesis"
canonical must define exactly one of path or legacy url for repo "anthesis"
canonical path must include an owner or namespace for repo "anthesis"
mirror legacy url must not contain credentials for repo "anthesis"
unsupported canonical provider "github" for repo "anthesis": only gitlab is supported
```

A missing `uid` is currently compatibility-supported and defaults to `id`; it is not a validation error. Explicit `uid` is recommended for stable cache and future history continuity.

Transport selection is not currently a configuration field, so an `unsupported transport` configuration error does not yet exist.

## Current runtime limitations

The examples in this document do not imply a general production repository control plane. Current behavior is limited to:

- GitLab canonical repositories
- exactly one GitHub or GitLab mirror
- default branch comparison and synchronization
- no tag or deleted-ref reconciliation
- default HTTPS resolution for path-based runtime operations
- built-in GitHub and GitLab bases only
- no serialized plan consumed directly by apply
- no durable execution journal
- no explicit ref-policy or approval model

See the ordered roadmap for the dependency path beyond this topology foundation.
