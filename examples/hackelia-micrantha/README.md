# Hackelia Micrantha organization example

`repora.yaml` declares the curated set of 50 repositories intended for Repora
management from the `hackelia-micrantha` GitHub organization as of 2026-08-24.

## Before use

1. Replace every `replace-with-gitlab-namespace` canonical namespace with the
   actual GitLab namespace. Nested GitLab paths are supported.
2. Confirm that each GitLab canonical repository exists and that its default
   branch represents the authoritative history.
3. Add Bitbucket Cloud mirrors where desired using the real
   `<workspace>/<repository>` mapping. Do not put credentials in the file.
4. Run `repoctl status -f examples/hackelia-micrantha/repora.yaml`.
5. Review `repoctl plan -f examples/hackelia-micrantha/repora.yaml --artifact`.
6. Use `repoctl apply ... --dry-run` before any real synchronization.

The GitLab placeholder paths are syntactically valid so the example remains
covered by Repora's strict configuration loader, but they are not operational
targets.

## Bitbucket Cloud mirrors

Bitbucket Cloud is supported as a mirror provider over HTTPS. The intended
per-repository form is:

```yaml
mirrors:
  - provider: github
    path: hackelia-micrantha/<repository>
  - provider: bitbucket
    path: <workspace>/<repository>
```

Bitbucket paths must contain exactly two segments: workspace and repository.
Repora resolves them at runtime to credential-free
`https://bitbucket.org/<workspace>/<repository>.git` URLs and delegates
authentication to system Git credential helpers.

Bitbucket remains mirror-only. Do not configure it as canonical, use a legacy
Bitbucket URL, embed an app password/token, or mislabel a Bitbucket location as
GitLab. Provider-administration posture metadata for Bitbucket remains explicit
`unavailable`; Git-derived reconciliation facts still use the shared mirror
pipeline.

## Scope

The topology is intentionally curated rather than a generated inventory of the
entire organization. Repora's current default-branch-only model does not encode
GitHub/Bitbucket visibility or default-branch names in this file; those are
observed where supported at runtime. Review the list before adopting it as
production topology, especially if repositories have been archived,
transferred, added, or deliberately excluded since the inventory date.
