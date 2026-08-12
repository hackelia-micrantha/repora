# ADR-0017: Bounded Managed Artifact Domain

Status: Accepted

Decision date: 2026-08-12

Last reviewed: 2026-08-12

Implemented by: pending issue #12

Related issues: #7, #12, #8, #22

## Context

Repora needs one controlled repository-content mutation domain to validate plan-before-apply behavior beyond Git-ref mirroring. README generation is useful and reviewable, but adding file mutation can easily expand into arbitrary filesystem writes, executable templating, hidden repository edits, or a premature universal mutation abstraction.

ADR-0010 already requires versioned, domain-specific plan artifacts until multiple implemented domains prove that a shared abstraction reduces real complexity. The current reconciliation artifact therefore remains Git-ref-only.

## Decision

Repora introduces a separate managed-artifact domain with these v1 boundaries:

- README is the only supported artifact type.
- The managed output path is fixed to repository-root `README.md`; configuration cannot select an arbitrary output path.
- A repository is unaffected unless README management is explicitly configured for that repository.
- Configuration grants Repora ownership only of the configured README artifact.
- Templates are local, configuration-root-relative inputs. Remote templates, absolute template paths, repository-escaping paths, and credential-bearing sources are forbidden.
- Rendering is deterministic, non-executable token replacement. V1 has no functions, conditionals, loops, includes, environment expansion, shell execution, plugin hooks, or recursive evaluation.
- Existing README state must be absent or a regular Git blob. Symlink, submodule, tree, or other non-regular entries at `README.md` fail planning rather than being silently replaced.
- Managed artifact plans use their own versioned artifact kind rather than extending `repora.io/reconciliation-plan`.
- Real apply requires a reviewed exact managed-artifact plan and revalidates repository, Git mode, and content preconditions before remote mutation.
- Managed artifact apply changes the canonical default branch only. Mirror reconciliation is always re-planned afterward as a separate operation.

## Configuration shape

The first implementation should use an explicit README field rather than a generic artifact list:

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
          summary: Deterministic repository mirror management
```

Known-field config decoding must reject unknown artifact types. Future artifact types add explicit configuration fields only after a separate issue or ADR defines their output path, input authority, plan contract, and stale-preflight semantics.

## Renderer boundary

V1 recognizes only exact placeholders:

- `{{repo.id}}`
- `{{repo.uid}}`
- `{{canonical.provider}}`
- `{{canonical.path}}`
- `{{value.<key>}}` for explicitly configured scalar values

Rendering is a single pass. Rendered values are data and are not interpreted again as templates. Unknown placeholders fail planning. Template and value line endings are normalized to LF before rendering so identical logical inputs do not depend on host checkout settings.

Template input and observed README content must be valid UTF-8 text without NUL bytes. The implementation must impose a bounded template/output size; v1 should use a conservative fixed limit suitable for README text rather than accepting unbounded input.

## Domain-specific plan artifact

The first managed artifact plan is conceptually:

```text
ManagedArtifactPlan {
  kind: repora.io/managed-artifact-plan
  version: 1
  repositories: RepositoryPlan[]
}

RepositoryPlan {
  uid: durable repository identity
  id: human-facing repository identity
  target: canonical provider + path + default branch
  base_oid: exact observed canonical head
  actions: ArtifactAction[]
}

ArtifactAction {
  type: WRITE_README
  path: README.md
  observed: present + git_mode + sha256
  desired: git_mode + sha256 + UTF-8 content
  template_sha256: sha256
  diff: deterministic unified text diff
}
```

For an existing regular README, desired Git mode is the exact observed regular-file mode so content management does not silently change executable state. A newly created README uses Git mode `100644`.

The artifact contains no template filesystem path, cache path, resolved transport URL, credential, environment value, author identity, or timestamp. The same topology, template bytes, values, canonical head, and observed README mode/content must produce byte-identical plan serialization.

## Plan and apply semantics

Planning is read-only:

1. load and validate configuration;
2. resolve the canonical repository;
3. observe the exact canonical default-branch OID;
4. inspect root `README.md` in that Git tree without modifying a user worktree;
5. if present, require a regular blob mode (`100644` or `100755`) and valid bounded UTF-8 text; preserve that mode as desired mode;
6. if absent, use desired mode `100644`;
7. render the desired README deterministically;
8. emit no action when content bytes are already equal;
9. otherwise emit exact mode/content preconditions, digests, desired content, and a deterministic unified diff.

Apply consumes the reviewed artifact rather than re-rendering the template. Before any remote mutation it must:

- validate artifact kind/version and repository identity;
- require the configured canonical provider/path and default branch to match the plan;
- require current canonical HEAD to equal `base_oid`;
- require current README presence, Git mode, and content digest to equal the observed state;
- require absent README state to remain absent;
- validate the desired mode, content digest, and fixed `README.md` path;
- fail closed before push on any stale or invalid input.

Execution must use isolated Git plumbing or an isolated temporary work area, never mutate an unrelated user checkout. The new commit must be a direct child of the reviewed base commit and update only root `README.md` content while preserving reviewed regular-file mode. Remote publication must retain an exact expected-head guard. Commit metadata is execution evidence, not part of deterministic planning.

Dry-run performs the same artifact validation and stale preflight but creates no commit and pushes nothing.

## Cross-domain ordering

Managed artifact changes and mirror reconciliation are deliberately not one transaction.

A successful README apply changes canonical HEAD, so any previously reviewed mirror reconciliation plan becomes stale by design. The required workflow is:

```text
artifact plan -> review -> artifact apply
                            |
                            v
                 fresh status/plan -> mirror apply
```

Repora must not silently rewrite or reuse an older mirror plan after content mutation.

## Evidence and recovery

Managed artifact results must expose enough evidence to explain the effect:

- plan digest;
- repository UID/ID;
- canonical provider/path/branch;
- reviewed base OID;
- resulting commit OID on success;
- observed and desired README Git mode and content digests;
- applied, stale, skipped, or failed outcome;
- sanitized error when applicable.

There is no automatic rollback. Recovery from stale input or failed publication is to inspect current canonical state and re-plan. A future durable journal format for this domain must remain domain-specific unless later evidence justifies a shared execution-record abstraction.

## Security implications

- No executable templates or plugins.
- No remote template fetching.
- No configurable output path in v1.
- No arbitrary file deletion or generation.
- No mutation when README management is absent.
- No replacement of symlink, submodule, tree, or non-text README state as if it were an ordinary text file.
- No persistence of local template/cache paths in plan artifacts.
- No credentials, tokenized URLs, environment values, or Git command lines in plans.
- Exact canonical-head, Git-mode, and content-digest preconditions fail closed on stale state.
- Apply never mutates a user checkout implicitly.

## Alternatives rejected

### Extend the reconciliation plan with file actions

Rejected. Git refs and managed files have different identity, comparison, stale-state, and execution semantics. ADR-0010 explicitly avoids a universal cross-domain plan without implemented evidence.

### Generic `artifacts: []` with configurable output paths

Rejected for v1. That becomes arbitrary file generation and weakens the ownership boundary.

### Go `text/template` or external template engines

Rejected for v1. Functions, control flow, includes, and extension hooks add unnecessary execution and determinism surface.

### Apply README and mirror updates in one plan

Rejected. Content apply changes canonical HEAD and invalidates mirror observations. Fresh re-planning keeps both domains exact and auditable.

## Consequences

### Positive

- README generation validates a second mutation domain without weakening Git-ref contracts.
- The managed path and template language are narrowly bounded.
- Plan review remains exact and deterministic.
- Existing repositories remain unchanged by default.
- Git file mode cannot change invisibly as a side effect of content management.
- Future artifact types require explicit design review.

### Costs

- README apply needs a new domain-specific planner/executor and plan schema.
- Operators perform artifact apply and mirror reconciliation as two explicit phases.
- The small renderer is intentionally less expressive than general template engines.
- Commit creation introduces Git content-mutation mechanics that require separate tests from mirror pushes.

## Future decision gate

Any additional managed artifact type must have a separate issue or ADR that specifies:

- fixed or otherwise bounded output ownership;
- allowed inputs and their trust boundary;
- deterministic renderer/serializer rules;
- exact observed/desired state and stale checks;
- mutation and recovery behavior;
- evidence requirements;
- why the existing README model is insufficient.

Arbitrary file generation, code generation, workflow/CI mutation, plugin execution, and remote template loading remain out of scope.
