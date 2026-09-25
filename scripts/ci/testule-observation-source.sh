#!/usr/bin/env bash
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
cd "$root"

output_dir="${1:-artifacts/testule}"
max_bytes="${TESTULE_OBSERVATION_MAX_BYTES:-1048576}"
package="repoctl/internal/plan"
target="TestReconcileIsDeterministicAndDoesNotMutateInputs"
stream="$output_dir/representative-go-test.json"
revision="$output_dir/subject-revision.txt"

mkdir -p "$output_dir"
umask 077
rm -f "$stream" "$revision"

set +e
go test -count=1 -json ./internal/plan -run "^$target\$" \
  | python3 ./scripts/ci/capture-json-stream.py "$stream" "$max_bytes"
pipeline_status=("${PIPESTATUS[@]}")
set -e

if [[ "${pipeline_status[0]}" -ne 0 ]]; then
  printf 'representative native Go target failed; refusing observation source\n' >&2
  exit 1
fi
if [[ "${pipeline_status[1]}" -ne 0 ]]; then
  printf 'representative native Go JSON capture failed; refusing partial/oversized source\n' >&2
  exit 1
fi

python3 ./scripts/ci/verify-go-test-observation.py "$stream" "$package" "$target"
git rev-parse HEAD > "$revision"

printf 'native observation source: %s %s @ %s\n' "$package" "$target" "$(cat "$revision")"
