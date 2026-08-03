# Reference synchronization policy

Status: Implemented by the ref-policy v1 slice

Repora's current reference policy is deliberately closed. It makes the existing runtime boundary explicit without adding branch, tag, or wildcard synchronization.

## Configuration

A repository may state the supported policy explicitly:

```yaml
policy:
  refs:
    version: 1
    scope: default-branch-only
    destructive: require-force
```

Omitting the policy produces the same effective values.

Configuration loading rejects unsupported policy versions, broader scopes, and permissive destructive modes. Repora does not accept future-looking policy values and then silently ignore them.

## Planning decisions

The planner maps the observed default-branch relationship to exact intent:

| State | Planned intent |
| --- | --- |
| `EQUAL` | no action |
| `BEHIND` | one normal default-branch push |
| `AHEAD` | one forced default-branch overwrite |
| `DIVERGED` | one forced default-branch overwrite |

Planning records destructive intent. A real mutation still requires explicit `--force`; dry-run and artifact review may describe a forced action without authorizing mutation.

## Enforcement boundaries

- configuration owns policy syntax, normalization, and supported-value validation;
- the planner owns relationship-to-action decisions;
- artifact validation and executor preflight remain defensive checks against topology, default-branch, force-intent, and stale-ref mismatches;
- Git force-with-lease remains remote-side defense in depth.

## Closed version-1 scope

Version 1 denies:

- non-default branches;
- tags;
- wildcard refspecs;
- deleted-ref reconciliation;
- permissive destructive modes;
- provider-side protected-branch API integration.

Protected-reference behavior is therefore simple in v1: only each remote's current default branch is eligible, and destructive reconciliation requires explicit command authorization.

## Compatibility

Existing configurations remain valid because omission normalizes to the closed v1 policy.

The current reconciliation artifact remains compatible because policy v1 has only one accepted interpretation. A future policy version with real alternatives must version or bind the exact artifact to the policy decision; it must not reinterpret a v1 artifact under broader rules.
