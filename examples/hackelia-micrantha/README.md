# Hackelia Micrantha organization example

`repora.yaml` declares every active repository visible in the
`hackelia-micrantha` GitHub organization on 2026-08-24.

## Before use

1. Replace every `replace-with-gitlab-namespace` canonical namespace with the
   actual GitLab namespace. Nested GitLab paths are supported.
2. Confirm that each GitLab canonical repository exists and that its default
   branch represents the authoritative history.
3. Run `repoctl status -f examples/hackelia-micrantha/repora.yaml`.
4. Review `repoctl plan -f examples/hackelia-micrantha/repora.yaml --artifact`.
5. Use `repoctl apply ... --dry-run` before any real synchronization.

The placeholder paths are syntactically valid so the example remains covered by
Repora's strict configuration loader, but they are not operational targets.

## Bitbucket boundary

Bitbucket cannot be added to this runnable configuration yet. The current
runtime accepts GitLab canonicals and GitHub or GitLab mirrors only. When a
Bitbucket adapter is implemented, the intended equivalent for each entry is:

```yaml
mirrors:
  - provider: github
    path: hackelia-micrantha/<repository>
  - provider: bitbucket
    path: <workspace>/<repository>
```

Until then, keep the Bitbucket mapping separately; do not mislabel Bitbucket
locations as GitLab or use legacy transport URLs to bypass provider validation.

## Scope

The inventory includes public, private, community, legacy, and organization
metadata repositories. Repora's current default-branch-only model does not
encode GitHub visibility or default-branch names in this file; those are
observed at runtime. Review the list before adopting it as production topology,
especially if repositories have been archived, transferred, or added since the
inventory date.
