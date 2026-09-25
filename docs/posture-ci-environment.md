# CI environment posture

Status: Current

`repoctl posture ci-environment OWNER/REPO` collects read-only evidence for the Micrantha flake-first CI / minimal-runner boundary without deciding compliance.

## Contract

The command emits `repora.posture-ci-environment` v2 JSON. Repora continues to accept v1 artifacts for compatibility. The serialized contracts are `schemas/posture-ci-environment-v1.schema.json` and `schemas/posture-ci-environment-v2.schema.json`.

The collector records:

- exact default-branch and commit evidence;
- `flake.nix` and `flake.lock` presence;
- GitHub Actions workflow discovery completeness;
- bounded per-workflow flake-invocation signals;
- bounded high-confidence imperative-install signals;
- bounded workload-tool invocation, known setup/provisioning, and conservative ambient-tool candidate signals;
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

Signals are observations only. V2 parses GitHub Actions YAML steps and recognizes a deliberately small workload-tool family set from the start of `run:` command lines. It separately records known setup actions and derives an ambient candidate only when a recognized workload tool lacks a corresponding recognized setup action in that workflow.

Commands beginning with `nix` are not reinterpreted as ambient nested tools, so `nix develop ... -c cargo test` does not become a bare Rust assumption. Comment-only lines are ignored. This is still bounded static evidence: shell wrappers, generated commands, PATH mutation, custom setup actions, composite actions, containers, and unusual syntax can remain unknown or require human classification. Runtime proof remains the responsibility of the runner/telemetry authority, such as Dubnium.

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
- undeclared ambient runner-tool assumptions are detectable only for the bounded v2 tool/setup patterns; arbitrary binaries and runtime PATH provenance remain outside static proof;
- Dubnium remains the runtime/space-time telemetry authority for its runner fleet.

Those remaining evidence gaps stay tracked by #197 rather than being hidden by the example profile.
