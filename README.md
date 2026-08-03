# Repora / repoctl

> Experimental, deterministic Git repository mirror control.

![status](https://img.shields.io/badge/status-pre--alpha-blue)
![license](https://img.shields.io/badge/license-BSL%201.1-orange)
![implementation](https://img.shields.io/badge/status-multi--mirror%20read-black)

## Overview

**Repora** manages repository state through explicit topology, observation, policy, exact planning, stale-safe execution, and durable evidence.

**repoctl** is the current Go CLI prototype. A repository entry has one GitLab canonical and one or more GitHub/GitLab mirrors. Status observes every mirror independently. Plan/apply/sync remain explicitly single-mirror until the exact multi-mirror execution contract is implemented.

Repora is pre-alpha. The broader repository-control-plane model is product direction, not a claim about current runtime capability.

## Implemented today

- strict YAML parsing and validation;
- multiple repository entries with bounded concurrency;
- durable `uid` identity separate from location;
- provider-relative `provider + path` topology;
- one or more unambiguous mirrors for status;
- stable mirror selectors such as `github:org/repository`;
- bounded single-mirror legacy URL compatibility;
- runtime HTTPS resolution for built-in GitHub and GitLab;
- local bare-cache preparation and system Git authentication;
- default-branch status states:
  - `EQUAL`
  - `BEHIND`
  - `AHEAD`
  - `DIVERGED`
  - `ERROR` for incomplete mirror observation;
- mirror-local status failure isolation;
- `repora.status` JSON version 2;
- closed ref-policy version 1:
  - default branch only;
  - destructive actions require explicit force authorization;
- deterministic exact reconciliation artifacts;
- `plan --artifact` export and `apply --plan-file` execution without re-planning;
- complete stale-ref preflight before mutation;
- normal behind pushes and explicit lease-protected overwrites;
- fail-closed immutable intent/result journal evidence;
- versioned human/JSON command contracts and nonzero failure status.

## Current limitations

- GitLab canonical repositories only;
- built-in GitHub/GitLab transport bases only;
- path-based operations use HTTPS by default;
- no user-selectable transport or provider bases;
- multi-mirror status is implemented, but multi-mirror plan/apply/sync are not;
- default branch only;
- no tags, wildcard refs, deleted-ref reconciliation, or complete ref inventory;
- no provider provisioning;
- no cross-remote transaction or rollback;
- no Anthesis policy integration;
- no supported release binaries or compatibility guarantee beyond committed versioned schemas.

## Current workflow

### Inspect one or more mirrors

```mermaid
flowchart LR
    Config[repora.yaml] --> Canonical[Observe canonical once]
    Canonical --> Mirrors[Observe each mirror]
    Mirrors --> Status[Status v2 results]
```

One mirror failure remains visible as `ERROR` and does not hide later mirrors. Operational failure returns exit `1`; complete ahead/diverged state returns `2`.

### Review and apply one mirror

```mermaid
flowchart LR
    Topology[Topology + ref policy] --> Observe[Single-mirror observation]
    Observe --> Planner[Deterministic planner]
    Planner --> Artifact[Exact plan artifact]
    Artifact --> Intent[Immutable INTENT]
    Intent --> Preflight[Complete stale preflight]
    Preflight --> Executor[Mutation or dry-run]
    Executor --> Result[Immutable RESULT]
```

Plan/apply/sync reject multi-mirror repositories before Git observation rather than choosing the first mirror implicitly.

## Configuration

Preferred form:

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
      - provider: gitlab
        path: micrantha-backup/anthesis
    mode: mirror
    policy:
      refs:
        version: 1
        scope: default-branch-only
        destructive: require-force
```

`id` is the operational label. `uid` is durable identity and should remain stable across renames, provider changes, moves, and transport changes.

When several mirrors are configured, each must use provider/path and targets must be unique. Array position is deterministic order, not identity.

Credentials must not be embedded in configuration. Authentication is delegated to system Git and credential helpers.

See [`docs/configuration/provider-path-topology-v1.md`](docs/configuration/provider-path-topology-v1.md).

## CLI

```bash
repoctl status -f repora.yaml
repoctl status -f repora.yaml --json
repoctl plan -f single-mirror.yaml
repoctl plan -f single-mirror.yaml --artifact > plan.json
repoctl apply -f single-mirror.yaml --dry-run
repoctl apply -f single-mirror.yaml --plan-file plan.json
repoctl apply -f single-mirror.yaml --plan-file plan.json --force
```

`sync` is currently an alias for `apply`.

Exit codes:

- `0`: success;
- `1`: operational, validation, stale, journal, execution, or output failure;
- `2`: complete unsafe status or missing authorization for a destructive real mutation.

## Contracts and architecture

- [Current architecture](docs/architecture/current-system.md)
- [Multi-mirror status](docs/architecture/multi-mirror-status.md)
- [Failure and recovery semantics](docs/architecture/failure-semantics.md)
- [Exact reconciliation artifact](docs/architecture/reconciliation-plan-artifact.md)
- [Execution journal](docs/architecture/execution-journal.md)
- [Closed ref policy](docs/architecture/ref-policy.md)
- [Status v2 migration](docs/cli/status-v2.md)
- [Active implementation plan](docs/plans/current.md)
- [Architecture decision index](docs/decisions/README.md)
- [Versioned schemas](schemas/)

## Development

Repora uses [mise](https://mise.jdx.sh/) for tool and task management.

```bash
mise install
mise run fmt
mise run lint
mise run test
mise run build
```

Direct Go tooling is also supported.

## Product direction

Repora is intended to become a local-first repository controller that is:

- **deterministic** — identical topology, observations, and policy produce the same exact plan;
- **reviewable** — mutation intent exists before execution;
- **stale-safe** — changed refs reject old plans;
- **auditable** — intent and outcome evidence is durable;
- **policy-driven** — unsupported or destructive behavior fails closed;
- **honest about partial failure** — no false atomicity or rollback claims.

Potential managed domains include Git ref reconciliation, deterministic repository artifacts, CI/security posture, repository evidence, and bounded document routing for AI-assisted work. They must reuse the plan, policy, execution, and evidence substrate rather than create parallel mutation paths.

## Security model

Current controls include:

- credentials delegated to system Git;
- credential-bearing HTTP URLs rejected;
- stable identity separated from transport;
- closed default-branch-only ref policy;
- destructive intent visible in the exact artifact;
- explicit `--force` authorization;
- full stale-ref preflight;
- force-with-lease defense in depth;
- fail-closed journal intent persistence;
- sanitized diagnostics and safe relative evidence references;
- strict denial of implicit multi-mirror mutation.

Future work includes exact multi-mirror target binding and outcomes, optional approvals/attestations, and supported release packaging.

## Roadmap

Immediate critical path:

1. exact multi-mirror artifacts and independent ordered apply;
2. per-mirror result and journal evidence with non-atomic recovery;
3. v0.1 release packaging, checksums, verification, and installation guidance.

The authoritative order is maintained in [`docs/plans/current.md`](docs/plans/current.md) and GitHub issues.

## License

Business Source License 1.1:

- free for personal and internal use;
- commercial or SaaS use requires a license;
- converts to Apache-2.0 on 2029-01-01.

See [LICENSE](LICENSE).

## Project status

Pre-alpha and actively evolving. Expect explicit version migrations and architecture refinement.

External contributions are currently closed while the core model stabilizes. Concrete use cases and failure reports are welcome.

Micrantha Software — [micrantha.com](https://micrantha.com)
