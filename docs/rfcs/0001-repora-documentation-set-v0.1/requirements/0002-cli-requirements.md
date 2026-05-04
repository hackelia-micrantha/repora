# REQ-0002: CLI Requirements

Status: Draft

## Commands

`repoctl` shall expose the following v0.1 command surface:

- `repoctl status`
- `repoctl plan`
- `repoctl apply`
- `repoctl sync`

## Output Requirements

- Human-readable output shall remain concise, stable, and suitable for
  interactive operator workflows
- Machine-readable JSON output shall be specified for automation and tests
- Output formats should distinguish observation, planned action, and execution
  result

## Flags

- `-f, --file <repora.yaml>`: path to the declarative repository specification
- `--json`: emit machine-readable JSON
- `--force`: permit destructive mirror overwrite from canonical state when
  divergence is detected
- `--parallel N` (future-ready): bound concurrent per-repository execution

## Exit Codes

- `0`: command completed successfully
- `1`: command failed due to configuration, authentication, authorization,
  network, or execution error
- `2`: divergence detected and execution refused because `--force` was not
  specified
