# CI environment posture v1

Status: Current

`repoctl posture ci-environment OWNER/REPO` collects read-only evidence for the Micrantha flake-first CI / minimal-runner boundary without deciding compliance.

## Contract

The command emits `repora.posture-ci-environment` v1 JSON. The serialized contract is `schemas/posture-ci-environment-v1.schema.json`.

The collector records:

- exact default-branch and commit evidence;
- `flake.nix` and `flake.lock` presence;
- GitHub Actions workflow discovery completeness;
- bounded per-workflow flake-invocation signals;
- bounded high-confidence imperative-install signals;
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

Signals are observations only. Shell comments, wrappers, generated commands, or unusual syntax can require human classification. Repora does not claim that static text inspection proves semantic ownership.

## Policy convergence

```bash
repoctl posture ci-environment OWNER/REPO > ci-environment.json
repoctl posture converge --ci-environment ci-environment.json > facts.json
repoctl posture report --profile POLICY.json --facts facts.json --as-of YYYY-MM-DD
```

Normalized facts use the `ci_environment.*` namespace. This preserves Repora's existing collection -> convergence -> policy/report boundary and avoids a second scanner or scoring model.

## Security boundary

The collector reuses the GET-only `GitHubReader` capability. It does not execute repository code, enter the flake, install tools, mutate provider state, write caches, or contact runner infrastructure. Workflow content and the optional profile are bounded untrusted data.
