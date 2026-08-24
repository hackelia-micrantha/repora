# GitHub posture inventory v1

Status: Current

`repoctl posture inventory` is Repora's first executable repository/CI posture slice. It is deliberately read-only and produces normalized evidence only; policy evaluation, findings, reports, scanner execution, remediation, and provider mutation remain separate later layers.

## Command

```bash
repoctl posture inventory OWNER/REPO
```

Public repositories can be inspected without a token. For private repositories or provider evidence requiring authenticated read access, Repora reads `GITHUB_TOKEN` and then `GH_TOKEN` from the environment. Tokens are never accepted as command-line arguments, persisted in the inventory, or included in evidence references.

The command emits `repora.posture-inventory` v1 JSON. [`../schemas/posture-inventory-v1.schema.json`](../schemas/posture-inventory-v1.schema.json) is the serialized contract.

## Fact states

Every normalized scalar/repository fact carries an evidence state:

| State | Meaning |
| --- | --- |
| `observed` | The source was available and the value was actually observed. Boolean `true` and `false` remain distinct observed values. |
| `unknown` | Evidence was readable but incomplete or dynamic enough that a reliable value cannot be asserted. |
| `unavailable` | The relevant provider evidence could not be read under current access, commonly HTTP 401/403/404. |

`false` is therefore never used as a substitute for missing permission or incomplete evidence.

A truncated Git tree illustrates the distinction: a file actually present in the returned tree is still `observed: true`; a file not present is `unknown` because the omitted portion could contain it.

## Repository facts

The GitHub-first inventory currently records:

- default branch;
- whether the default branch is protected;
- required status-check names where branch-protection detail is available;
- required approving review count;
- force-push protection;
- branch-deletion protection;
- `CODEOWNERS` presence in supported GitHub locations;
- `SECURITY.md` presence in supported GitHub locations;
- root license presence;
- issue-template presence, excluding the issue-chooser `config.yml`/`config.yaml` control file;
- pull-request-template presence;
- dependency-update configuration for Dependabot and common Renovate entrypoints;
- GitHub Actions workflow paths.

GitHub-recognized repository paths are evaluated with provider-appropriate case sensitivity. Pull-request template filenames use GitHub's documented case-insensitive filename behavior while their containing repository paths remain case-sensitive.

Repository metadata, branch metadata, detailed branch protection, Git tree state, and workflow blobs are separate evidence sources. Failure to read detailed branch protection does not suppress file-backed facts when repository contents remain readable.

## GitHub Actions facts

Each discovered `.github/workflows/*.yml` or `.yaml` file is parsed as untrusted data without executing it. The inventory records:

- workflow-level declared permissions;
- job-level declared permissions;
- use of `pull_request_target`;
- literal or mapped `runs-on` labels;
- `self_hosted_label`, which reports only whether the literal `self-hosted` label is observed; it does **not** claim what runner infrastructure actually executes a job;
- dynamic or group-only runner selection as `unknown` for `self_hosted_label` when literal label evidence is unavailable;
- `uses:` references from steps and reusable-workflow jobs;
- whether an external action is third-party relative to GitHub-maintained `actions/*` / `github/*` and the inspected repository;
- pinning evidence as `immutable-sha`, `immutable-digest`, `mutable-ref`, `unversioned`, or `local`.

Pinning classification is an observation, not a policy judgment. For example, a tag and a branch are both represented as `mutable-ref` because determining the remote ref type is outside this bounded inventory slice.

Workflow normalization is bounded to 1 MiB per file. Oversized or malformed workflow content is represented with `unknown` workflow state rather than parsed or treated as a passing workflow. Unreadable workflow blobs are `unavailable`.

`workflows_state` describes workflow-path discovery completeness from the Git tree; each workflow also carries its own content-normalization state.

## Read-only capability boundary

The collector depends on a `GitHubReader` interface that exposes only:

- repository reads;
- branch reads;
- branch-protection reads;
- Git-tree reads;
- blob reads.

The production HTTP adapter issues only `GET` requests. There is no provider mutation method available to the inventory collector, which makes the no-mutation property structural rather than conventional.

Repository contents are read through immutable Git tree/blob evidence for the observed default-branch commit. No checkout, local file mutation, scanner execution, issue creation, pull-request creation, branch-protection update, or other provider write is performed.

## Failure semantics

Authentication/authorization/not-found responses (`401`, `403`, `404`) are represented as unavailable provider evidence because GitHub can intentionally conceal private-resource existence with `404`.

Transport failures, malformed provider responses, oversized provider responses, and unexpected server status codes remain operational errors and cause a nonzero command exit rather than being silently converted into posture facts.

## Next layer

`repora.posture-inventory` v1 owns repository/CI fact collection only. The implemented documentation, mirror, hooks/local-workflow, and bounded commit-history domains reuse the shared fact/provenance model. Offline posture policy and deterministic reporting consume those normalized facts after source artifacts validate.

Those layers must not reclassify unavailable evidence as failure/success or bypass this observation-versus-policy boundary.
