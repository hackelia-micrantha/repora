# Hooks and local-workflow posture

`repoctl posture hooks OWNER/REPO` emits the versioned `repora.posture-hooks` v1 fact artifact for a GitHub repository.

The collector is deliberately read-only. It reads repository metadata, the default-branch tree, selected hook/config/document blobs, and GitHub Actions workflow files. It never installs, sources, invokes, or bootstraps hook code and never runs package-manager or repository-owned commands.

## Fact model

All observations reuse the shared posture `Fact[T]` states:

- `observed`: the value was established from bounded evidence;
- `unknown`: the value is representable but cannot be established from current evidence;
- `unavailable`: relevant evidence could not be read under current access.

The artifact reports default branch/commit identity, optional profile presence, detected/declared hook manager, configured entrypoints, static network-load signals, required local checks and observable CI coverage, bootstrap documentation, and bypass/escape-hatch documentation.

Executable state remains `unknown` for GitHub tree observations in v1 because the current shared reader does not expose tree mode. Existence is not treated as proof that a hook is executable or trusted.

## Manager detection

The baseline detector recognizes:

- `.pre-commit-config.yaml` as pre-commit;
- `lefthook.yml` / `lefthook.yaml` as Lefthook;
- `.husky/*` hook files as Husky;
- `.githooks/*` files as a generic/custom hook path.

A repository may declare additional expectations in `.repora/posture-hooks.yaml`:

```yaml
kind: repora.posture-hooks-profile
version: 1
manager: custom
hook_paths:
  - .githooks/pre-commit
required_checks:
  - gofmt
  - go test
bootstrap_docs:
  - docs/development.md
bypass_docs:
  - docs/hooks-bypass.md
```

The profile is bounded data, not executable policy. Paths must be normalized repository-relative paths and the total number of observation targets is limited.

## CI relationship

`required_checks` are local-workflow expectations. For each declared check, v1 records whether its text is observable in GitHub Actions workflow files. This is evidence of coverage, not proof of semantic equivalence. CI remains the enforcement source; local hooks are early feedback unless later policy explicitly says otherwise.

## Trust signals

Hook/config blobs are inspected only as bounded text. `network_loaded=true` is emitted when static inspection sees common network-loading signals such as `curl`, `wget`, or HTTP(S) references. This does not execute or validate the remote content; it gives #121 normalized evidence from which policy can later produce findings.

A missing bootstrap or bypass document is represented as an observed `false` only when a declared path can be proven absent from a complete tree. Truncated or inaccessible evidence stays `unknown` or `unavailable` rather than becoming a healthy conclusion.

## Security boundary

The hooks collector does not:

- install or modify Git hooks;
- execute repository scripts, generated hook code, package managers, or hook-manager binaries;
- download code referenced by hooks;
- claim that a present hook is safe, executable, or enforced;
- replace CI or policy evaluation;
- mutate repository/provider settings.
