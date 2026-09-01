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

Roles such as `canonical` or `staging` and trust domains such as `private`, `agent`, `public`, or `recovery` are topology/policy facts. They are not inferred from provider names and do not independently grant mutation authority.

## Relationship semantics

### Mirror

Represents the same permitted Git state on another endpoint. Existing reconciliation/ref-policy semantics remain authoritative.

### Projection

Produces a deterministic derived repository representation across a disclosure/trust boundary.

```text
private source revision
  -> bounded projection profile
  -> isolated materialization
  -> inventory/digests/diff
  -> exact plan
  -> target repository update
```

Projection is deliberately not `git push --mirror`. Private history and metadata do not cross the boundary by default.

### Promotion

Moves one exact candidate from staging into a more authoritative endpoint.

```text
staging candidate
  -> exact promotion plan
  -> optional policy authorization
  -> stale-safe canonical mutation
```

Promotion binds source candidate, target base, topology/profile identity, and plan digest. It never treats staging state as authoritative merely because it exists.

### Public contribution import

Imports an exact public PR/commit/patch into private development as a candidate proposal. Import itself has no canonical mutation authority. Any accepted result proceeds through a fresh private-side promotion/review cycle.

### Archive

Produces a recovery-oriented representation, initially Git bundles plus a versioned manifest and integrity digest. Archive is not a writable mirror and does not imply that forge metadata, secrets, or databases are protected.

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

A future CanForge endpoint is represented as `provider: forgejo` with a distinct installation identity. Canonical source-of-truth migration is a separate project decision; provider capability must not cause it automatically.

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
2. Define the topology/relationship ADR and versioned schema (#150).
3. Add generic Forgejo installations (#151).
4. Generalize canonical endpoints (#152).
5. Implement deterministic projection (#153), using Dubnium #536 as the first pilot.
6. Implement staging promotion (#154), integrating Sandcastle/Anthesis only through existing ownership boundaries.
7. Implement public contribution import (#155).
8. Add archive/recovery evidence (#156).
9. Add overlay semantics only when a concrete consumer exists.

## Security invariants

- No relationship implies its inverse.
- Provider identity does not imply trust role.
- Topology facts do not grant authority.
- Credentials and transport URLs are not durable repository identity.
- Plan/apply remains exact and stale-safe.
- Changed candidate/source/target/profile/policy-relevant state invalidates prior authority.
- Projection never copies private history by default.
- Public state never gains automatic reverse-sync authority.
- External policy authorization is additive and cannot weaken Repora's local safety model.
- Recovery claims are based on verified archive contents, not the existence of another checkout or same-host mirror.

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