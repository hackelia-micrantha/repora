# Bounded commit-history posture v1

Status: Current when merged with #122

## Purpose

`repoctl posture commits OWNER/REPO` emits read-only repository/process evidence for a bounded window of the GitHub default-branch commit history.

The artifact kind is `repora.posture-commits` version 1. The optional repository observation profile is `repora.posture-commits-profile` version 1 at `.repora/posture-commits.yaml`.

This domain collects facts for later policy evaluation. It does not decide whether unsigned commits, large commits, sensitive-path changes, merge shapes, or missing pull-request associations are acceptable.

## Default observation

When a complete default-branch tree proves that no profile exists, the built-in profile uses:

- history limit: 20 commits;
- file-count threshold: 50 files;
- changed-lines threshold: 1000 additions plus deletions;
- no sensitive-path patterns;
- commit-to-pull-request association enabled.

The history limit is capped at 50 commits. Sensitive-path configuration is capped at 128 repository-relative `path.Match` patterns. `*` does not cross `/`, so `.github/workflows/*` matches workflow files while `.github/*` does not match nested workflow paths.

Example profile:

```yaml
kind: repora.posture-commits-profile
version: 1
history_limit: 20
sensitive_paths:
  - SECURITY.md
  - .github/workflows/*
  - internal/auth/*
file_count_threshold: 50
changed_lines_threshold: 1000
inspect_pull_requests: true
```

These values configure observation only. They do not assign severity or define compliance.

## Facts

For each observed commit, v1 can record:

- commit SHA;
- whether it has multiple parents;
- GitHub signature verification state (`verified`, `unsigned`, `unverified`, or `unknown`);
- additions plus deletions;
- observed changed-file count;
- whether the returned file list is complete;
- deterministic file-count and changed-lines threshold crossings;
- configured sensitive paths found in the observed file list;
- associated pull-request count when that endpoint is enabled and accessible.

The inventory also records the default branch/commit, profile presence, configured observation limits/thresholds, and whether the requested history window was truncated.

## Conservative evidence rules

The shared posture states remain authoritative:

- `observed` means the provider evidence established a value;
- `unknown` means the fact is representable but the current scope/evidence cannot establish it;
- `unavailable` means relevant evidence could not be read under current access.

GitHub commit detail can return at most the bounded file page used by v1. When the file list may be incomplete, a positive sensitive-path match or already-exceeded file threshold remains observable, but a negative conclusion becomes `unknown`.

A zero associated-pull-request count does **not** prove that a change bypassed review or was pushed directly to the default branch. Consequently `direct_to_default_branch` and `unreviewed_change` remain explicit `unknown` facts in v1. A later provider adapter may populate them only when evidence proves the relationship.

Tag signatures and release-boundary facts are also explicit `unknown` in v1 because this collector does not inspect tag/release evidence.

## Privacy and governance boundary

Commit posture is repository/process-risk evidence, not people analytics. The v1 artifact does not serialize author or committer identity fields and does not produce:

- productivity scores;
- performance metrics;
- identity profiles;
- blame assignments;
- inferred developer intent.

Signing requirements, sensitive-path expectations, threshold severity, and remediation belong to the later policy layer (#121).

## Security boundary

Collection uses the same GET-only GitHub HTTP capability used by the other GitHub posture domains. Optional `GITHUB_TOKEN` or `GH_TOKEN` values are runtime inputs and are not serialized.

Repository-owned profile data is bounded, strictly parsed, and never executed. Commit posture does not mutate provider settings, repositories, branches, tags, releases, or history.

## Contracts

- `schemas/posture-commits-v1.schema.json`
- `schemas/posture-commits-profile-v1.schema.json`

Consumers should key on both `kind` and `version`.
