# Assumption-free repository bootstrap

Status: Current

Repora implements the Micrantha repository-bootstrap v1 contract without turning observed repository state into design authority.

## Authority and contract pin

The organization-owned contract remains authoritative in `hackelia-micrantha/.github`. Repora pins bootstrap semantics to merge `af0ec6581e7593b4e5cb8a5ada4294cde86115e8`, including:

- `resolved`, `unresolved`, and `not-applicable` decision states;
- unique decision keys and key-specific resolved-value semantics;
- provenance as an attributable claim that must be independently verified before a dependent effect;
- observations as evidence only, never decision authority;
- exact plan binding and stale revalidation of authority-bearing inputs.

Repora does not copy or redefine the authoritative manifest schema. The proposal emitted by `bootstrap init` references the immutable organization schema URL.

Bootstrap write confinement requires Go 1.24 or newer; repository CI and Nix packaging currently use Go 1.25.

## CLI lifecycle

```text
repoctl bootstrap init [--root DIR]
repoctl bootstrap inspect [--root DIR] [--json]
repoctl bootstrap plan [--root DIR] [--manifest FILE] [--artifact]
repoctl bootstrap apply --plan-file FILE [--json]
repoctl bootstrap discover -f repora.yaml [--previous FILE] OWNER/REPO
```

`init` creates `.repora/bootstrap.proposed.json` with create-only semantics. It does not select language, license, visibility, CI, release, topology, or provider settings. The organization default branch `main` is the only organization-policy resolution in the default proposal.

`inspect` is local and read-only. It records bounded file/workflow observations and a deterministic snapshot identity.

`plan` consumes an exact proposal and binds its absolute input path and digest, the absolute repository root, local snapshot, pinned organization contract, authority claims, executable actions, blocked actions, unresolved decision keys, and explicit CI applicability. Every executable action digest must equal the bound manifest digest.

`apply` accepts only an exact bootstrap-plan v1 artifact. V1 can create only `.repora/bootstrap.json` from the reviewed proposal. It revalidates the proposal digest, snapshot, authority-claim set, parent directory, and create-only target before writing. Proposal and manifest writes use Go's root-confined filesystem API, so symlink/path traversal cannot redirect effects outside the selected repository root. Stale evidence exits with status 2. It does not create providers, licenses, implementation scaffolds, release configuration, or CI.

## CI applicability

Bootstrap intentionally separates whether executable CI applies from how applicable CI is implemented.

`delivery.ciProvider` is the only v1 decision used to derive CI applicability:

- resolved -> `applicable`;
- not-applicable -> `not-applicable`;
- missing or unresolved -> `unresolved`.

Observed workflows, `flake.nix`, or `flake.lock` never resolve CI applicability.

When CI is explicitly applicable, the Micrantha CI/CD standard requires a repository-owned Nix flake to own project-specific CI/build/test tooling. Bootstrap v1 reports the required follow-up but does not generate a workflow or a placeholder flake.

## Discovery

`bootstrap discover` is GET-only with respect to GitHub. It records:

- registered or unregistered status relative to the supplied Repora configuration;
- exact default-branch commit/tree identity;
- bootstrap-manifest presence and validity;
- explicit CI applicability from a valid manifest only;
- flake, lock, license, and workflow observations;
- provider-inherited Micrantha community-health baseline versus local overrides;
- a deterministic fingerprint.

Supplying `--previous` classifies the scan as `first-discovery`, `unchanged`, or `changed`. Core discovery does not require a scheduler, webhook, or provider mutation capability; those triggers may be layered on later.

## Trust boundary

Persisting a bootstrap manifest stores provenance claims as data. It does not authorize or exercise the decisions inside it. Future mutation capabilities must verify applicable human/project/policy authority independently before using those decisions.

Provider creation remains outside bootstrap v1. Automatic discovery never mutates a repository merely because it is new or non-conformant.