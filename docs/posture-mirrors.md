# Mirror posture v1

Status: Current

`repoctl posture mirrors -f repora.yaml` emits deterministic mirror-management facts for repositories already declared in Repora topology.

The output contract is `repora.posture-mirrors` v1. It reuses the shared posture `observed`, `unknown`, and `unavailable` evidence states and the existing mirror reconciliation semantics instead of implementing a second drift algorithm.

## Command

```bash
repoctl posture mirrors -f repora.yaml > mirror-posture.json
```

The command loads the normal Repora configuration. Canonical and mirror identities therefore come from the same `provider + path` topology used by status, plan, and apply.

## Observation boundary

Mirror posture v1 reuses `status.CheckAll` to refresh and inspect Repora's local bare mirror cache. That observation step can create or update files under Repora's local cache and can configure/fetch cache remotes. It does **not** push to canonical or mirror repositories, synchronize a mirror, publish releases, or mutate provider settings.

The collector records:

- repository ID and durable UID;
- declared `mirror` mode;
- synchronization direction `canonical_to_mirror`;
- explicit canonical `provider:path` identity;
- explicit mirror `provider:path` identities;
- local cache remote names;
- canonical and mirror default-branch names where observable;
- canonical and mirror default-branch commit evidence;
- default-branch-name drift;
- the existing `EQUAL`, `BEHIND`, `AHEAD`, `DIVERGED` reconciliation state;
- ahead/behind commit counts;
- provider visibility and current-actor push-permission facts when a provider adapter exposes them;
- tag and release drift fields, currently `unknown` in v1.

## Drift semantics

Commit drift is not recalculated independently. Mirror posture projects the same comparison used by Repora status:

- `EQUAL` — canonical and mirror default branches resolve to the same history;
- `BEHIND` — the mirror lacks canonical commits;
- `AHEAD` — the mirror contains commits absent from canonical;
- `DIVERGED` — both sides contain unique commits;
- unavailable evidence — the mirror comparison could not be established safely.

Default-branch-name drift is a separate fact. A mirror can have equal commit evidence while declaring a different default branch name; the posture artifact preserves that distinction.

Missing or inaccessible evidence never becomes an observed drift value. If either branch name cannot be established, `default_branch_drift` is `unknown` or `unavailable` rather than `false`, unless an independent provider read establishes the branch name.

## Provider metadata

The provider metadata boundary is intentionally narrower than repository mutation authority.

For GitHub endpoints, mirror posture uses the existing GET-only HTTP transport against repository metadata. When returned by GitHub it normalizes:

- `default_branch`;
- `visibility`;
- the authenticated/current actor's `permissions.push` value.

If GitHub omits actor permissions, `current_actor_push_permission` remains `unknown`; omission is not interpreted as `false`. A `401`, `403`, or `404` remains `unavailable` under the shared provider evidence semantics.

GitLab provider-administration metadata is not implemented in mirror posture v1, so those facts are `unavailable` with explicit unsupported-provider evidence. This does not affect local Git reconciliation evidence when the repository can still be fetched.

The fact name is `current_actor_push_permission`, not generic `writeable`, because provider authorization depends on the authenticated identity and does not prove that every actor can write.

## Tags and releases

Repora's current reconciliation contract is default-branch-only. Mirror posture v1 therefore includes `tag_drift` and `release_drift` as representable fact fields but emits them as `unknown` with scope evidence.

This preserves the schema needed by later policy/reporting work without silently expanding mirror reconciliation into tags, wildcard refs, releases, or publishing behavior.

## Evidence and failure behavior

Mirror posture preserves the shared fact states:

- `observed` — a value was actually established;
- `unknown` — the contract can represent the fact but the current adapter/scope cannot determine it;
- `unavailable` — evidence could not be read under the current access or observation failed.

Per-mirror failures remain attached to that mirror. A failed mirror must not cause a healthy mirror to be reported as drifted or healthy by inference.

Operational failures in canonical observation or malformed configured topology cause a nonzero command result rather than a partially fabricated repository result.

## Security boundary

Mirror posture v1:

- does not call push operations;
- does not invoke mirror synchronization;
- does not publish or delete refs/releases;
- does not change branch protection or repository visibility;
- does not assign severity or produce policy findings;
- does not turn current-actor push permission into a universal repository-writeability claim.

Provider credentials remain delegated to Git/credential helpers for fetches and to `GITHUB_TOKEN` / `GH_TOKEN` for optional GitHub API reads. Credentials are not serialized into posture evidence.

## Relationship to other posture work

`repora.posture-mirrors` v1 is the authoritative mirror-management fact contract. The offline policy/reporting layer consumes it rather than re-reading repository topology or re-running its own divergence algorithm.

Hooks/local-workflow and bounded commit/process facts remain separate observation domains.
