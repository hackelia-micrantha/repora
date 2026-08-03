# Status JSON v2

`repoctl status --json` emits:

```json
{
  "kind": "repora.status",
  "version": 2,
  "repos": []
}
```

Version 2 adds stable multi-mirror target identity and mirror-local failure reporting.

## Mirror shape

```json
{
  "target": "github:org/repository",
  "provider": "github",
  "path": "org/repository",
  "ref": "HEAD",
  "commit": "abc1234",
  "state": "BEHIND",
  "ahead": 0,
  "behind": 3
}
```

A failed mirror retains the same identity and uses `state: "ERROR"` with an `error` field. Commit evidence may be empty when the failing operation occurred before it was resolved.

## Migration from v1

Version 1 represented one mirror and did not include stable target/path fields or an explicit error state.

Consumers should:

1. validate `kind == "repora.status"`;
2. branch on `version`;
3. iterate every `repos[].mirrors[]` entry;
4. use `target` as the stable selector in v2;
5. treat `ERROR` as incomplete observation, not divergence;
6. never infer target identity from array position.

Historical schema: `schemas/cli-status-v1.schema.json`.

Current schema: `schemas/cli-status-v2.schema.json`.

## Exit status

JSON output does not replace process status:

- `0`: all mirrors observed and equal/behind;
- `1`: any repository or mirror observation is incomplete;
- `2`: observation is complete and at least one mirror is ahead or diverged.
