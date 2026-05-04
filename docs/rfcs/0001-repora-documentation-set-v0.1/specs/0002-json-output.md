# SPEC-0002: JSON Output

Status: Draft

## `status --json`

```json
{
  "repos": [
    {
      "id": "payments-api",
      "canonical": {
        "ref": "HEAD",
        "commit": "abc1234"
      },
      "mirrors": [
        {
          "provider": "github",
          "ref": "HEAD",
          "commit": "def5678",
          "state": "BEHIND",
          "ahead": 0,
          "behind": 3
        }
      ]
    }
  ]
}
```

## `plan --json`

```json
{
  "plan": [
    {
      "id": "payments-api",
      "actions": [
        {
          "type": "PUSH_MIRROR",
          "target": "github",
          "behind": 3,
          "destructive": false
        }
      ]
    }
  ]
}
```

## State Values

- `EQUAL`: canonical and mirror refs resolve to equivalent history
- `BEHIND`: mirror lacks commits reachable from canonical
- `AHEAD`: mirror contains commits not reachable from canonical
- `DIVERGED`: canonical and mirror both contain unique commits
- `UNKNOWN`: state could not be determined due to configuration, auth, network,
  or execution failure
