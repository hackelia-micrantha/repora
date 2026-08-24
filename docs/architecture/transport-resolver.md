# Transport Resolver

## Status

Implemented as the runtime boundary between durable provider/path identity and Git transport.

## Ownership

Declarative endpoints identify repositories with:

- `provider`
- provider-relative `path`

The transport resolver owns conversion of that identity into a Git remote. Resolved URLs are runtime state and must not become durable repository identity.

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
      - provider: bitbucket
        path: micrantha/anthesis
```

At runtime, the default HTTPS resolver produces:

- `https://gitlab.com/micrantha/anthesis.git`
- `https://github.com/hackelia-micrantha/anthesis.git`
- `https://bitbucket.org/micrantha/anthesis.git`

GitHub and GitLab also have built-in SSH bases. Bitbucket Cloud support is intentionally HTTPS-only in this slice so authentication remains delegated to system Git credential helpers without adding app-password, token, or OAuth lifecycle state to Repora.

## Provider matrix

| Provider | Canonical | Mirror | HTTPS | SSH | Provider-admin posture |
| --- | --- | --- | --- | --- | --- |
| GitLab | yes | yes | yes | yes | unavailable in posture v1 |
| GitHub | no | yes | yes | yes | GET-only metadata supported |
| Bitbucket Cloud | no | yes | yes | no | unavailable in posture v1 |

Bitbucket Server/Data Center and custom provider bases are not supported.

## Bitbucket Cloud path boundary

Bitbucket mirrors must use provider/path form with exactly:

```text
<workspace>/<repository>
```

Nested GitLab-style namespaces are not accepted for Bitbucket. Schemes, credentials, traversal, query/fragment delimiters, backslashes, colons, leading/trailing slashes, and other malformed workspace/repository paths fail closed before Git transport is configured.

## Compatibility

GitHub and GitLab endpoints may temporarily provide `url` instead of `path` where legacy compatibility still applies. This is a bounded compatibility path:

- `path` and `url` are mutually exclusive.
- Legacy URLs are not authoritative identity.
- Credential-bearing URLs are rejected.
- New configuration should use provider-relative paths.
- Bitbucket mirrors do **not** accept legacy URLs; they require provider/path identity.

## Safety

Configuration validation checks provider presence, endpoint ambiguity, role support, and provider-relative path shape. URL construction happens only when status processing prepares Git remotes. Resolver errors identify provider, path, and transport without including credentials.

Bitbucket runtime URLs are credential-free. Credentials remain external to `repora.yaml`, status, plans, execution records, journals, and diagnostics.

Custom provider base URLs, Bitbucket SSH, Bitbucket Server/Data Center, and removal of remaining GitHub/GitLab legacy URL support are separate future work.
