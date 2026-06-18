# SPEC-0001: `repora.yaml` Schema (v0.1)

Status: Draft

```yaml
repos:
  - id: string
    canonical:
      provider: string # hint only
      url: string
    mirrors:
      - provider: string # hint only
        url: string
    mode: mirror # fixed in v0.1
```

## Rules

- `id` must be unique and stable across provider renames or URL changes
- `mode` must be `mirror` in v0.1
- `canonical.url` must identify an existing, reachable Git remote
- Each `mirrors[].url` must identify an existing, reachable Git remote
- `provider` is a human and implementation hint, not an authority boundary
- Credentials must not be stored in this file

## Explicit Defaults

Unless overridden by a future global configuration file:

- Cache directory: `~/.cache/repora`
- Sync mode: `mirror`
- Divergence behavior: fail closed
- Concurrency: sequential execution
