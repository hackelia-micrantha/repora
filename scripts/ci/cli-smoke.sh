#!/usr/bin/env bash
set -euo pipefail

binary="${1:-./bin/repoctl}"

"$binary" --help >/dev/null

assert_subcommand_help() {
  local command="$1"
  local output
  local status

  set +e
  output="$("$binary" "$command" --help 2>&1)"
  status=$?
  set -e

  if [ "$status" -ne 1 ]; then
    printf '%s --help exited %d, want 1\n' "$command" "$status" >&2
    printf '%s\n' "$output" >&2
    return 1
  fi
  if [ -z "$output" ]; then
    printf '%s --help produced no output\n' "$command" >&2
    return 1
  fi
}

assert_subcommand_help status
assert_subcommand_help plan
assert_subcommand_help apply
