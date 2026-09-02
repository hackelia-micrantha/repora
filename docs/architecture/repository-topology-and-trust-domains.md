# Repository topology and trust domains

Status: **Proposed** under #149 / #150. This document records the target architecture and must not be interpreted as implemented behavior until the corresponding contracts land.

## Problem

Repora's current executable core is intentionally strong around mirror reconciliation: stable repository identity, explicit topology, exact plan artifacts, stale preflight, ref policy, partial-failure evidence, and provider-neutral runtime transport. That model should remain intact.

New use cases require relationships that are not mirrors:

- a local multi-repository workspace (#147);
- an agent staging repository promoted into a private canonical repository;
- a private canonical repository projected into a clean public/community repository;
- public contributions imported into private development as proposals;
- recovery archives distinct from writable mirrors;
- future public-base/private-overlay composition.

Treating every relationship as a mirror would grant the wrong mutation semantics and blur trust/disclosure boundaries.

## Target model

```text
Project
  -> logical Repository
      -> hosted Endpoint(s)
      -> explicit Role / Trust domain
      -> explicit directed Relationship(s)
```

### Logical repository

Durable repository identity independent of hosting provider and transport.

### Endpoint

One hosted representation of a logical repository.

Conceptually:

```yaml
provider: forgejo
instance: canforge
repository: hackelia-micrantha/dubnium
```

The exact schema is owned by #150.

### Provider instance

Self-hostable providers require explicit installation identity. `CanForge` and `Dubnium Forgejo` should be Forgejo instances, not bespoke provider types.

### Role and trust domain

Roles such as `canonical` or `staging` and trust domains such as `private`, `agent`, `public`, or `recovery` are descriptive topology/policy facts. They are not inferred from provider names and do not independently grant read, write, projection, promotion, merge, or other mutation authority.

A label such as `trusted`, `canonical`, `internal`, `private`, or `agent` must never be treated as a sufficient authorization predicate. Trust-domain classification may constrain policy; it cannot mint authority.

For example, this is forbidden in principle:

```text
if endpoint.trustDomain == "private" then allowWrite
```

Authority remains a separate concern enforced through Repora's exact local safety model and, where configured, an additive external authorization decision.

## Relationship semantics

### Mirror

Represents the same permitted Git state on another endpoint. Existing reconciliation/ref-policy semantics remain authoritative.

### Projection

Produces a deterministic derived repository representation across a disclosure/trust boundary.

```text
private source revision/tree
  -> bounded projection profile
  -> isolated materialization
  -> target-tree identity + inventory/digests/diff
  -> exact plan
  -> target repository update
```

Projection is deliberately not `git push --mirror`. Private history and metadata do not cross the boundary by default.

Projection binds the source and resulting materialized tree independently:

```yaml
source:
  revision: abc123
  tree_digest: sha256:SOURCE
projection:
  profile_digest: sha256:PROFILE
materialization:
  tree_digest: sha256:TARGET_TREE
  inventory_digest: sha256:INVENTORY
target:
  expected_revision: def456
plan_digest: sha256:PLAN
```

The exact schema remains owned by #153. The invariant is that the reviewed effect binds `SOURCE + PROFILE -> TARGET_TREE`; approval of the source revision alone is insufficient.

Projection mappings must also classify whether later public contribution import has an explicit deterministic inverse. Generated, many-to-one, one-to-many, lossy, or ambiguous mappings are non-invertible by default.

### Promotion

Moves one exact candidate from staging into a more authoritative endpoint.

```text
staging candidate
  -> exact promotion plan
  -> optional policy authorization
  -> stale-safe canonical mutation
```

Promotion binds source candidate, target base, topology/profile identity, and plan digest. It never treats staging state as authoritative merely because it exists or carries an authority-suggestive role/trust-domain label.

### Public contribution import

Imports an exact public PR/commit/patch into private development as a candidate proposal. Import itself has no canonical mutation authority. Any accepted result proceeds through a fresh private-side promotion/review cycle.

When the public repository is a projection, Repora must use the exact projection profile/mapping that produced the public surface. A forward mapping never implies its inverse.

Conceptually:

```yaml
mappings:
  - from: docs/public/
    to: docs/
    import:
      strategy: reversible

  - generated_from:
      - metadata/book.yaml
      - templates/README.md
    to: README.md
    import:
      strategy: prohibited
```

Automatic import is allowed only for explicitly reversible deterministic mappings. Generated, lossy, transformed, many-to-one, one-to-many, unknown, or ambiguous mappings fail closed for automatic import. A non-importable public change may still be preserved as a review proposal without mutating private source state.

### Archive

Produces a recovery-oriented representation, initially Git bundles plus a versioned manifest and integrity digest. Archive is not a writable mirror and does not imply that forge metadata, secrets, or databases are protected.

Archive evidence must describe coverage explicitly. For example:

```yaml
coverage:
  git:
    objects: complete
    refs: complete
    lfs: unsupported
  forge:
    issues: unsupported
    pull_requests: unsupported
    packages: unsupported
    actions: unsupported
    database: unsupported
```

The exact representation may use richer states such as `complete`, `partial`, `unsupported`, or `indeterminate`, but it must distinguish **Git repository recoverability** from **forge/application recoverability**.

### Workspace

Local checkout composition remains separately owned by #147. Workspace relationships do not become build dependencies, hosted endpoints, or mirror relationships.

### Overlay

Deferred until a concrete consumer exists. A likely future use is public core + private additions, but projection/promotion must be proven before adding another mutation/composition domain.

## Reference topology: Dubnium

```text
Sandcastle / agent workspace
       |
       v
Dubnium Forgejo staging
       |
       | promotion (#154)
       v
private canonical endpoint
       |
       +---- mirror/archive (#156) ----> recovery storage
       |
       | projection (#153)
       v
hackelia-micrantha/dubnium-community
       ^
       |
 public PR/commit
       |
       +---- import proposal (#155) ----> private review -> promotion
```

A future CanForge endpoint is represented as `provider: forgejo` with a distinct installation identity. Canonical source-of-truth migration is a separate project decision; provider capability must not cause it automatically. Vanilla Forgejo remains the architectural compatibility target; CanForge is an installation/integration target rather than a dependency of the model.

## Ownership

### Repora

Owns:

- logical repository and endpoint identity;
- relationship configuration and validation;
- observation/planning;
- deterministic materialization where applicable;
- exact stale-safe execution;
- execution evidence.

Does not own:

- legal/IP/disclosure policy;
- agent scheduling/workspace runtime;
- Forgejo hosting/service operations;
- authorization policy semantics.

### Dubnium / hosting system

Owns Forgejo service runtime, network exposure, storage, credentials, backup scheduling, and agent execution infrastructure.

### Anthesis

May optionally authorize the exact consequential Repora effect at `pre_apply`. An Anthesis allow cannot weaken Repora local policy, stale checks, force/lease requirements, or plan binding.

### Sandcastle

May provide exact checkpoint/workspace/repository revision facts. These are evidence/state dependencies, not authority.

## Credential boundary

Desired staging topology:

```text
agent runtime:
  staging write       yes
  canonical write     no
  public write        no by default

Repora effector:
  staging read        bounded
  canonical write     exact planned effect only
  public write        exact projection effect only

Anthesis:
  repository facts    read/evaluate only
  repository writes   none
```

The runtime/network implementation must enforce the claimed boundary; configuration alone is not complete mediation.

## Compatibility and migration

Current Repora mirror configurations should remain valid. #150/#152 must define a migration such that existing canonical/mirror topology maps into the richer model without changing current executable behavior.

Do not migrate source-of-truth hosts automatically. Configuration/schema migration and hosting migration are separate decisions.

## Sequencing

1. Finish `v0.2.0` release (#138).
2. Define the topology/relationship ADR and versioned schema (#150), including `trustDomain != authority`.
3. Generalize canonical endpoints (#152).
4. Develop generic Forgejo installation support (#151) in parallel where useful; vanilla Forgejo must remain sufficient for conformance.
5. Implement deterministic projection (#153) first, using Dubnium #536 as the initial pilot. This proves exact source observation, clean materialization, materialized-tree identity, plan/apply, stale-target checks, and evidence at lower mutation risk.
6. Implement staging promotion (#154). Its provider-neutral model depends on #150/#152; the Dubnium Forgejo reference pilot additionally depends on #151.
7. Implement public contribution import (#155) after #153/#154 so explicit invertibility semantics are available and accepted candidates still enter through a separate promotion cycle.
8. Add archive/recovery evidence (#156) independently after #150.
9. Add overlay semantics only when a concrete consumer exists.

This ordering is implementation guidance, not semantic coupling. Projection and promotion remain distinct relationships and neither grants authority to the other.

## Security invariants

- No relationship implies its inverse.
- Provider identity does not imply trust role.
- Role/trust-domain classification does not grant authority.
- Topology facts do not grant authority.
- Credentials and transport URLs are not durable repository identity.
- Plan/apply remains exact and stale-safe.
- Projection binds source tree, projection profile, materialized target tree, target/base state, and plan independently.
- Changed candidate/source/target/materialization/profile/policy-relevant state invalidates prior authority.
- Projection never copies private history by default.
- Public state never gains automatic reverse-sync authority.
- A forward projection never implies a reverse mapping; ambiguous/generated mappings are non-importable by default.
- External policy authorization is additive and cannot weaken Repora's local safety model.
- Recovery claims are based on verified archive contents and explicit coverage, not the existence of another checkout or same-host mirror.

## Related

- #149 — repository topology/trust-domain epic.
- #150 — domain model/ADR.
- #151 — Forgejo instances.
- #152 — provider-neutral canonical endpoints.
- #153 — projection.
- #154 — promotion.
- #155 — public contribution import.
- #156 — archive/recovery.
- #147 — workspace manifests/bootstrap.
- #30 — optional Anthesis policy integration.