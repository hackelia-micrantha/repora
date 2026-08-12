#!/usr/bin/env bash
set -euo pipefail

binary="${1:-./bin/repoctl}"

"$binary" --help >/dev/null
version_output="$("$binary" --version)"
if [[ ! "$version_output" =~ ^repoctl\ .+\ \(.+\)$ ]]; then
  printf '%s --version produced unexpected output: %q\n' "$binary" "$version_output" >&2
  exit 1
fi

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

"$binary" validate-report --help >/dev/null
"$binary" validate-report ./examples/repository-assessment-v1.json >/dev/null
"$binary" list-findings --help >/dev/null
"$binary" list-findings ./examples/repository-assessment-v1.json >/dev/null
"$binary" generate-scorecard --help >/dev/null
"$binary" generate-scorecard ./examples/repository-assessment-v1.json >/dev/null

assessment_tmpdir="$(mktemp -d)"
trap 'rm -rf "$assessment_tmpdir"' EXIT
"$binary" assess --help >/dev/null
"$binary" assess "$assessment_tmpdir/assessment.json" >/dev/null
"$binary" validate-report "$assessment_tmpdir/assessment.json" >/dev/null
