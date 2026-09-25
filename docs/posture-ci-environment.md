# CI environment posture

Status: Current

`repoctl posture ci-environment OWNER/REPO` collects read-only evidence for the Micrantha flake-first CI / minimal-runner boundary without deciding compliance.

## Contract

The command emits `repora.posture-ci-environment` v2 JSON. The v2 contract is `schemas/posture-ci-environment-v2.schema.json`. Offline convergence continues accepting the unchanged v1 contract in `schemas/posture-ci-environment-v1.schema.json`; v1 does not contain host-tool or setup-action signals.

The collector records:

- exact default-branch and commit evidence;
- `flake.nix` and `flake.lock` presence;
- GitHub Actions workflow discovery completeness;
- bounded per-workflow flake-invocation signals;
- bounded high-confidence imperative-install signals;
- bounded YAML-aware direct host-tool invocation signals;
- bounded explicit tool setup/provisioning action signals;
- an optional project declaration at `.repora/posture-ci-environment.yaml`;
- explicitly declared irreducible `bootstrap` or `platform` external inputs.

It does **not** infer that CI is applicable merely because a workflow exists, and it does not treat the presence of a flake as proof that CI actually uses it.

## Optional project declaration

```yaml
kind: repora.posture-ci-environment-profile
version: 1
ci_applicability: applicable
external_inputs:
  - id: macos-xcode
    class: platform
    rationale: Xcode is a vendor-controlled macOS platform capability
```

`ci_applicability` is one of `applicable`, `not-applicable`, or `unresolved`. A missing, unreadable, or malformed declaration remains unknown/unavailable evidence rather than becoming a default.

External inputs are declarations, not exemptions granted by Repora. Offline policy decides whether an external input is acceptable for a repository.

## Workflow signals

Flake invocation signals are deliberately conservative and currently recognize:

- `nix flake check`;
- explicit current-flake `nix build .` / `.#...`;
- explicit current-flake `nix develop .` / `.#...`;
- explicit current-flake `nix run .` / `.#...`.

Imperative-install signals cover high-confidence shell installation patterns such as apt/apk/dnf/yum/brew, pip, global npm/pnpm/yarn installs, cargo/go installs, and rustup toolchain/component installation.

V2 additionally parses workflow YAML as data and inspects only `run` and `uses` scalar values for the new signals. Direct host-tool observation uses a bounded command-position set: Go, Node/package managers, Python/pip, Rust, Java build tools, Docker/Podman, Kubernetes/Helm, Terraform/OpenTofu, jq/yq, and gh. Explicit setup-action observation recognizes a bounded set such as `actions/setup-go`, `actions/setup-node`, `actions/setup-python`, `actions/setup-java`, selected Rust/tooling actions, and explicit Docker/Terraform/Kubernetes/Helm setup actions.

A recognized current-flake command line such as `nix develop .#ci --command go test ./...` is not also labeled as a direct host-tool invocation. Comments and scalar text outside `run` / `uses` are excluded from these v2 signals. If YAML cannot be parsed, the v2 signals are `unknown`; unreadable or oversized workflows remain `unavailable` / `unknown` rather than becoming observed empty sets.

Signals are observations only. A direct `go test` signal does not prove whether Go came from the runner image, a setup action, a container, or another boundary. Shell comments, wrappers, generated commands, or unusual syntax can require human classification. Repora does not claim that static text inspection proves semantic ownership.

## Policy convergence

```bash
repoctl posture ci-environment OWNER/REPO > ci-environment.json
repoctl posture converge --ci-environment ci-environment.json > facts.json
repoctl posture report --profile POLICY.json --facts facts.json --as-of YYYY-MM-DD
```

Normalized facts use the `ci_environment.*` namespace. This preserves Repora's existing collection -> convergence -> policy/report boundary and avoids a second scanner or scoring model.

## Security boundary

The collector reuses the GET-only `GitHubReader` capability. It does not execute repository code, enter the flake, install tools, mutate provider state, write caches, or contact runner infrastructure. Workflow content and the optional profile are bounded untrusted data.


## Micrantha flake-first example policy and fixtures

Repora includes an illustrative external policy profile at `examples/posture/micrantha-flake-first-policy-v2.json` plus deterministic collector fixtures under `cmd/repoctl/testdata/flake-first/`.

The example is a **consumer of** the Micrantha organization policy; it is not a replacement authority for `.github#125/#126`. It demonstrates four currently supportable conditional expectations when CI is explicitly applicable:

- `flake.nix` is present;
- `flake.lock` is present;
- at least one observed workflow has a bounded current-flake invocation signal;
- no observed workflow has a bounded high-confidence imperative-install signal.

The fixture matrix covers conformant, violating, explicit N/A, declared platform-input evidence, and unresolved/ambiguous cases through the actual `posture converge` -> `posture report` CLI boundary.

Important limits remain:

- a flake file or workflow signal does not by itself prove that the flake is the authoritative environment;
- static workflow signals are bounded observations, not full shell semantics;
- declared `platform` / `bootstrap` inputs remain visible evidence and do not automatically become policy exceptions;
- bounded direct host-tool and setup-action signals now expose statically visible runner/tool assumptions, but they do not prove provenance and are not a complete ambient-binary inventory;
- Dubnium remains the runtime/space-time telemetry authority for its runner fleet.

Those remaining evidence gaps stay tracked by #197 rather than being hidden by the example profile.
