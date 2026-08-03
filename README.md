# Repora / repoctl

> Experimental, deterministic Git repository mirror control.

![status](https://img.shields.io/badge/status-pre--alpha-blue)
![license](https://img.shields.io/badge/license-BSL%201.1-orange)
![implementation](https://img.shields.io/badge/status-multi--mirror%20apply-black)

## Overview

**Repora** manages repository state through explicit topology, observation, policy, exact planning, stale-safe execution, honest partial results, and durable evidence.

**repoctl** is the current Go CLI prototype. A repository entry has one GitLab canonical and one or more GitHub/GitLab mirrors. Status, planning, dry-run, and real apply/sync support multiple mirrors.

Repora is pre-alpha. The broader repository-control-plane model is product direction, not a claim about current runtime capability.

## Implemented today

- strict YAML validation and durable `uid` identity;
- provider-relative canonical and mirror topology;
- one or more GitHub/GitLab mirrors with stable `provider:path` identity;
- bounded single-mirror legacy URL compatibility;
- runtime HTTPS resolution and system Git authentication;
- independent per-mirror `EQUAL`, `BEHIND`, `AHEAD`, `DIVERGED`, and `ERROR` status;
- status v2 with mirror-local failure isolation;
- closed ref-policy v1: default branch only and explicit destructive authorization;
- exact reconciliation artifact v2 across every required mirror action;
- target rebinding by provider/path rather than serialized alias or position;
- complete topology, policy, branch, and expected-OID preflight before action zero;
- sequential independent mirror execution in artifact order;
- continuation to later mirrors after a runtime push failure;
- normal pushes and reviewed force-with-lease overwrites;
- apply v3 per-target before/desired/after/outcome evidence;
- execution-record v3 immutable intent/result evidence;
- bounded repository concurrency;
- historical artifact v1, apply v2, and execution-record v1/v2 compatibility.

## Current limitations

- GitLab canonical repositories only;
- built-in GitHub/GitLab HTTPS transport bases only;
- default branch only;
- mirrors execute sequentially inside one repository;
- no tags, wildcard refs, deleted-ref reconciliation, or complete ref inventory;
- no provider provisioning;
- no cross-remote transaction or automatic rollback;
- no Anthesis policy integration;
- no supported release binaries yet.

## Execution model

```mermaid
flowchart LR
    Config[Topology + policy] --> Observe[Observe canonical and mirrors]
    Observe --> Artifact[Exact artifact v2]
    Artifact --> Prepare[Prepare all selected repositories]
    Prepare --> Intent[Immutable INTENT]
    Intent --> Preflight[Validate every expected OID]
    Preflight --> Execute[Sequential independent pushes]
    Execute --> Result[Apply v3 + immutable RESULT]
```

Mirror identity is provider/path. Configuration order determines deterministic review and execution order but is not identity. Runtime aliases are local details and cannot retarget imported intent.

After complete preflight, one mirror failure does not prevent later independent mirrors. A valid result may be:

```text
APPLIED, FAILED, APPLIED
```

Successful earlier mirrors are not rolled back. The command returns nonzero and retry requires a fresh status observation and new artifact.

## Configuration

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

Credentials must not be embedded in configuration. Authentication is delegated to system Git and credential helpers.

See [`docs/configuration/provider-path-topology-v1.md`](docs/configuration/provider-path-topology-v1.md).

## CLI

```bash
repoctl status -f repora.yaml
repoctl status -f repora.yaml --json

repoctl plan -f repora.yaml
repoctl plan -f repora.yaml --artifact > plan.json

repoctl apply -f repora.yaml --dry-run
repoctl apply -f repora.yaml --plan-file plan.json --dry-run --json

repoctl apply -f repora.yaml --plan-file plan.json
repoctl apply -f repora.yaml --plan-file plan.json --force --json
```

`sync` is currently an alias for `apply`.

Output compatibility:

- single-mirror-only selections retain `repora.apply` v2;
- mixed or multi-mirror selections use `repora.apply` v3.

Exit codes:

- `0`: success;
- `1`: operational failure, stale preflight, journal failure, or partial success;
- `2`: complete destructive intent requires `--force`.

## Contracts and architecture

- [Current architecture](docs/architecture/current-system.md)
- [Failure and recovery semantics](docs/architecture/failure-semantics.md)
- [Exact reconciliation artifact](docs/architecture/reconciliation-plan-artifact.md)
- [Execution journal](docs/architecture/execution-journal.md)
- [Apply v3 migration](docs/cli/apply-v3.md)
- [Architecture decisions](docs/decisions/README.md)
- [Active implementation plan](docs/plans/current.md)
- [Versioned schemas](schemas/)

## Development

```bash
mise install
mise run fmt
mise run lint
mise run test
mise run build
```

Direct Go tooling is also supported.

## Security model

Current controls include:

- credentials delegated to system Git;
- credential-bearing HTTP URLs rejected;
- durable identity separated from transport and runtime aliases;
- closed default-branch-only ref policy;
- reviewed destructive intent plus explicit command authorization;
- complete all-target stale-ref preflight;
- force-with-lease for every forced action;
- fail-closed intent persistence;
- path-bound per-target result evidence;
- sanitized diagnostics and safe relative journal references;
- no implicit target selection, replay, rollback, or atomicity claim.

## Roadmap

The mirror-controller implementation path is complete. The immediate critical path is v0.1 release packaging, checksums, packaged-binary smoke coverage, verification, and installation guidance.

The authoritative order is maintained in [`docs/plans/current.md`](docs/plans/current.md) and GitHub issues.

## License

Business Source License 1.1:

- free for personal and internal use;
- commercial or SaaS use requires a license;
- converts to Apache-2.0 on 2029-01-01.

See [LICENSE](LICENSE).

## Project status

Pre-alpha and actively evolving. Expect explicit version migrations until v0.1 packaging is complete.

External contributions are currently closed while the core model stabilizes. Concrete use cases and failure reports are welcome.

Micrantha Software — [micrantha.com](https://micrantha.com)
