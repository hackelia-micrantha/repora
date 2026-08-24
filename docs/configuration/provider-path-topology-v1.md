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
      - provider: bitbucket
        path: micrantha/anthesis
    mode: mirror
    policy:
      refs:
        version: 1
        scope: default-branch-only
        destructive: require-force
```

The runtime derives credential-free HTTPS remotes immediately before Git operations. Authentication remains delegated to system Git and credential helpers.

## Provider matrix

| Provider | Canonical role | Mirror role | Runtime transport | Path shape |
| --- | --- | --- | --- | --- |
| GitLab | supported | supported | HTTPS or SSH | `namespace[/subgroup...]/repository` |
| GitHub | not supported | supported | HTTPS or SSH | `owner/repository` |
| Bitbucket Cloud | not supported | supported | HTTPS only | `workspace/repository` |

Bitbucket Server/Data Center and custom provider bases are outside v1.

## Fields

### `repos`

A non-empty list. Repository entries may be processed concurrently through `--parallel` after all selected preparation and force authorization succeeds.

### `id` and `uid`

`id` is the CLI label. `uid` is durable identity. An omitted `uid` defaults to `id`, but an explicit stable UID is recommended. Both must be unique.

### `canonical`

The authoritative source. The current runtime supports GitLab canonical repositories only.

### `mirrors`

One or more GitHub, GitLab, or Bitbucket Cloud targets.

- status, exact planning, dry-run, and real mutation support every configured mirror through the shared reconciliation pipeline;
- every mirror in a multi-mirror entry must use provider/path form;
- Bitbucket mirrors always require provider/path form, including when they are the only mirror;
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
micrantha/repora
```

Bitbucket Cloud is stricter than GitLab: its path must contain exactly two segments, `workspace/repository`. Query/fragment delimiters, colons, backslashes, leading/trailing slashes, and nested namespace forms fail closed.

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

A single GitHub/GitLab mirror entry may continue to use legacy URLs. Exactly one of `path` or `url` is required per endpoint, and credential-bearing HTTP URLs are rejected.

Legacy URLs are compatibility transport input, not identity. Multi-mirror entries require provider/path so every target is unambiguous. Bitbucket mirrors do not accept legacy URLs. New exact multi-mirror artifacts and execution evidence never persist transport URLs.

## Execution semantics

Provider/path topology is used consistently across:

- status v2 target identity;
- reconciliation artifact v2 actions;
- current runtime alias rebinding;
- apply v3 source and target results;
- execution-record v3 intent/result evidence.

Before action zero, Repora verifies current configuration, status, policy, default branches, and all expected OIDs. After preflight, mirror actions execute sequentially in exact artifact order. Runtime failure of one mirror does not prevent later independent targets, and successful earlier targets are not rolled back.

Bitbucket targets use the same comparison, exact-plan, stale-preflight, partial-success, and force-with-lease semantics as other mirrors. There is no provider-specific execution bypass.

## Migration

1. Add and preserve an explicit `uid`.
2. Replace endpoint URLs with provider-relative paths.
3. Add additional mirrors only after every target has unambiguous provider/path identity.
4. For Bitbucket Cloud, add `provider: bitbucket` with `<workspace>/<repository>`; do not embed credentials or use a transport URL.
5. Review `repoctl status` and `repoctl plan --artifact`.
6. Run `apply --dry-run` before real mutation.
7. Treat partial apply results as evidence requiring fresh status and planning, not replay or rollback.

## Current runtime boundary

Implemented:

- GitLab canonical repositories;
- one or more GitHub/GitLab/Bitbucket Cloud mirrors;
- Bitbucket Cloud HTTPS transport with external Git credential handling;
- stable provider/path identity through status, planning, execution, results, and journals;
- default-branch-only closed ref policy;
- sequential independent mirror mutation with force-with-lease;
- runtime HTTPS resolution;
- bounded GitHub/GitLab single-mirror legacy URL compatibility.

Not implemented:

- Bitbucket canonical repositories;
- Bitbucket SSH, Server, or Data Center;
- tags, non-default branches, wildcard refs, or deleted-ref reconciliation;
- provider provisioning;
- custom provider bases or user-selectable transport;
- concurrent mirror mutation;
- automatic rollback or hosted control-plane behavior.
