# Transport Resolver

## Status

Implemented as the first runtime slice of issue #16. The provider/path schema documentation and broader migration remain tracked by issue #19.

## Ownership

Declarative endpoints identify repositories with:

- `provider`
- provider-relative `path`

The transport resolver owns conversion of that identity into an HTTPS or SSH Git remote. Resolved URLs are runtime state and must not become durable repository identity.

```yaml
repos:
  - id: anthesis
    uid: repo.anthesis
    canonical:
      provider: gitlab
      path: micrantha/anthesis
    mirrors:
      - provider: github
        path: hackelia-micrantha/anthesis
```

At runtime, the default HTTPS resolver produces:

- `https://gitlab.com/micrantha/anthesis.git`
- `https://github.com/hackelia-micrantha/anthesis.git`

The SSH resolver produces:

- `git@gitlab.com:micrantha/anthesis.git`
- `git@github.com:hackelia-micrantha/anthesis.git`

## Compatibility

An endpoint may temporarily provide `url` instead of `path`. This is a bounded legacy compatibility path:

- `path` and `url` are mutually exclusive.
- Legacy URLs are not authoritative identity.
- Credential-bearing URLs are rejected.
- New configuration should use provider-relative paths.

## Safety

Configuration validation checks provider presence, endpoint ambiguity, and provider-relative path shape. URL construction happens only when status processing prepares Git remotes. Resolver errors identify provider, path, and transport without including credentials.

Custom provider base URLs, explicit transport selection in user configuration, and removal of legacy URL support are follow-up work coordinated with issue #19.
