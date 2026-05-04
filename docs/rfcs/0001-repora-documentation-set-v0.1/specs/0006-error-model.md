# SPEC-0006: Error Model

Status: Draft

## `AUTH_ERROR`

Authentication or authorization failure while accessing a remote, including SSH
failure, invalid credentials, insufficient permissions, or rejected push
authorization.

Exit code: `1`

## `NETWORK_ERROR`

Failure to reach a remote due to DNS resolution, timeout, unavailable network,
TLS failure, or interrupted transport.

Exit code: `1`

## `DIVERGENCE`

Canonical and mirror histories both contain unique commits or refs such that
automatic safe reconciliation is not possible under default behavior.

Exit code: `2` unless `--force` is specified

## `CONFIG_ERROR`

Invalid or incomplete configuration, including schema violation, duplicate
repository IDs, missing required fields, unsupported mode, or malformed remote
URL.

Exit code: `1`

## `EXECUTION_ERROR`

Unexpected local command failure not otherwise classified, including unavailable
`git`, corrupted local cache, failed ref resolution, or unsupported repository
state.

Exit code: `1`
