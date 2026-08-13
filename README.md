# Repora / repoctl

> Experimental, deterministic Git repository mirror and managed-artifact control.

![status](https://img.shields.io/badge/status-pre--alpha-blue)
![license](https://img.shields.io/badge/license-BSL%201.1-orange)
![implementation](https://img.shields.io/badge/status-managed%20README%20apply-black)

## Overview

**Repora** manages repository state through explicit topology, observation, policy, exact planning, stale-safe execution, honest partial results, and durable evidence.

**repoctl** is the current Go CLI prototype. A repository entry has one GitLab canonical and one or more GitHub/GitLab mirrors. Status, planning, dry-run, and real apply/sync support multiple mirrors. Repora also supports opt-in deterministic management of the canonical root `README.md` as its first managed artifact type.

Repora is pre-alpha. The broader repository-control-plane model is product direction, not a claim that arbitrary repository artifacts or provider operations are implemented.

## Name

**Repora** is a coined name combining **repo**—the common shorthand for a source-code repository—with the botanical cadence of **flora**. It reflects the project’s role in managing a collection or ecosystem of repositories while fitting the broader Micrantha naming family.

## Why Go

Go was chosen for `repoctl` because it fits the operational shape of a repository-control CLI:

- it produces small, self-contained binaries that are straightforward to distribute across Linux, macOS, Windows, CI runners, and administrative hosts;
- fast startup and modest runtime overhead suit commands that inspect many repositories and frequently invoke system Git;
- the standard library provides strong support for subprocess control, cancellation, timeouts, filesystem work, structured encoding, and bounded concurrency without requiring a large dependency graph;
- static typing, explicit error handling, built-in testing, formatting, vetting, and race detection support deterministic behavior and long-term maintenance;
- cross-compilation and reproducible release automation keep the packaging and deployment boundary simple.

The choice is pragmatic rather than ideological. Repora delegates Git protocol and credential behavior to the installed Git executable instead of reimplementing Git, while Go owns the topology, policy, planning, validation, execution, and evidence contracts. Languages such as Rust could provide stronger compile-time guarantees for some internal states, but Go currently offers the better trade-off for implementation speed, operational simplicity, contributor accessibility, and the project's expected performance envelope.

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
- opt-in `artifacts.readme` configuration with configuration-root-contained local templates;
- deterministic, bounded, non-executable one-pass README rendering;
- exact `repora.io/managed-artifact-plan` v1 review artifacts with observed/desired mode and content digests;
- canonical Git-tree README observation without a worktree checkout;
- exact managed README stale preflight and mutation-free dry-run;
- isolated candidate commit construction that can change only root `README.md`;
- guarded canonical README push using the reviewed base OID as an exact Git lease;
- managed-artifact INTENT/RESULT journal evidence and apply-result v1 output, including partial-success reporting;
- bounded repository concurrency;
- versioned Linux, macOS, and Windows release packaging with SHA-256 checksums;
- vulnerability, CodeQL, Git-history secret, dependency-license, and workflow-policy validation;
- historical artifact v1, apply v2, and execution-record v1/v2 compatibility.

## Current limitations

- GitLab canonical repositories only;
- built-in GitHub/GitLab HTTPS transport bases only;
- default branch only;
- mirrors execute sequentially inside one repository;
- managed repository artifacts currently support only root `README.md`;
- managed README templates are local, bounded, and deliberately non-executable;
- managed README apply does not automatically mutate mirrors; mirror propagation requires a fresh separate status/plan/apply review cycle;
- no arbitrary file, CI/workflow, or docs-site generation;
- no tags, wildcard refs, deleted-ref reconciliation, or complete ref inventory;
- no provider provisioning;
- no cross-remote transaction or automatic rollback;
- no Anthesis policy integration;
- no package-manager distribution, release signing, or full provenance attestation.

## Installation

Version tags publish archives for Linux amd64, macOS amd64/arm64, and Windows amd64. Each release includes `checksums.txt`; packaged binaries report the embedded tag and source commit through `repoctl --version`.

See [release installation and verification](docs/release.md) for target support, checksum commands, installation, local reproduction, and rollback.

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

Managed README mutation uses the same safety philosophy but a separate mutation domain and review artifact. It follows:

```text
plan-readme → review exact managed plan → apply-readme dry-run/preflight → journal INTENT → isolated commit → fresh preflight → exact-base leased canonical push → journal RESULT
```

After a successful README apply, any earlier mirror observation/plan is stale. Mirror propagation begins again with fresh `status`, then a separately reviewed mirror `plan`, then mirror `apply`.

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

Managed README configuration is opt-in:

```yaml
artifacts:
  readme:
    template: templates/README.md.tmpl
    values:
      summary: A deterministic README managed by Repora.
```

Template paths are resolved relative to the configuration file and must remain within that configuration root. See the complete [`examples/managed-readme/`](examples/managed-readme/) example.

Credentials must not be embedded in configuration. Authentication is delegated to system Git and credential helpers.

See [`docs/configuration/provider-path-topology-v1.md`](docs/configuration/provider-path-topology-v1.md).

## CLI

Mirror reconciliation:

```bash
repoctl --version
repoctl status -f repora.yaml
repoctl status -f repora.yaml --json

repoctl plan -f repora.yaml
repoctl plan -f repora.yaml --artifact > plan.json

repoctl apply -f repora.yaml --dry-run
repoctl apply -f repora.yaml --plan-file plan.json --dry-run --json

repoctl apply -f repora.yaml --plan-file plan.json
repoctl apply -f repora.yaml --plan-file plan.json --force --json
```

Managed README review/apply:

```bash
repoctl plan-readme -f repora.yaml
repoctl plan-readme -f repora.yaml --artifact > readme-plan.json

repoctl apply-readme -f repora.yaml --plan-file readme-plan.json --dry-run
repoctl apply-readme -f repora.yaml --plan-file readme-plan.json
repoctl apply-readme -f repora.yaml --plan-file readme-plan.json --json
```

`sync` is currently an alias for mirror `apply`.

Output compatibility:

- single-mirror-only selections retain `repora.apply` v2;
- mixed or multi-mirror selections use `repora.apply` v3;
- managed README plans use `repora.io/managed-artifact-plan` v1;
- managed README execution journals use `repora.io/managed-artifact-execution-record` v1;
- managed README real-apply JSON uses `repora.io/managed-artifact-apply-result` v1.

Mirror-command exit codes:

- `0`: success;
- `1`: operational failure, stale preflight, journal failure, or partial success;
- `2`: complete destructive intent requires `--force`.

Managed README apply exit codes:

- `0`: success or no managed README changes;
- `1`: invalid input, operational/journal failure, or partial remote success;
- `2`: reviewed managed-artifact plan is stale.

## Contracts and architecture

- [Current architecture](docs/architecture/current-system.md)
- [Failure and recovery semantics](docs/architecture/failure-semantics.md)
- [Exact reconciliation artifact](docs/architecture/reconciliation-plan-artifact.md)
- [Execution journal](docs/architecture/execution-journal.md)
- [Managed artifact architecture](docs/architecture/managed-artifacts.md)
- [Managed README planning and execution](docs/architecture/managed-artifact-planning.md)
- [Apply v3 migration](docs/cli/apply-v3.md)
- [Release installation and verification](docs/release.md)
- [Release checklist](docs/release-checklist.md)
- [Security CI and finding triage](docs/security-ci.md)
- [Benchmark scope](docs/benchmarks.md)
- [Changelog](CHANGELOG.md)
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
- reviewed destructive mirror intent plus explicit command authorization;
- complete all-target stale-ref preflight;
- force-with-lease for every forced mirror action;
- fail-closed intent persistence;
- path-bound per-target result evidence;
- managed README desired content bound by exact digest/mode and review diff;
- managed README authority limited to root `README.md` with contained local templates;
- managed README INTENT persisted before candidate-object creation or remote mutation;
- fresh managed README stale preflight immediately before exact-base leased canonical push;
- no automatic reuse of pre-artifact mirror observations/plans after canonical mutation;
- sanitized diagnostics and safe relative journal references;
- no implicit target selection, replay, rollback, or atomicity claim;
- tag-only release publication with least-privilege workflow permissions;
- reachable-vulnerability, CodeQL, secret, dependency-license, and workflow-policy gates.

## Roadmap

The mirror-controller implementation, v0.1.0 release boundary, routing/context foundations, repository assessment contracts, and first managed-artifact implementation are complete in code.

Future managed-artifact types remain separate design decisions; README support is not a generic arbitrary-file generator. Package-manager distribution, signing/provenance hardening, and broader provider/ref coverage remain future work.

Anthesis policy integration is explicitly deferred and is not part of the current execution path.

The authoritative order is maintained in [`docs/plans/current.md`](docs/plans/current.md) and GitHub issues.

## License

Business Source License 1.1:

- free for personal and internal use;
- commercial or SaaS use requires a license;
- converts to Apache-2.0 on 2029-01-01.

See [LICENSE](LICENSE).

## Project status

Pre-alpha and actively evolving. The v0.1.0 release has been published and independently smoke-verified; expect explicit contract migrations while the broader control-plane model continues to stabilize.

External contributions are currently closed while the core model stabilizes. Concrete use cases and failure reports are welcome.

Micrantha Software — [micrantha.com](https://micrantha.com)
