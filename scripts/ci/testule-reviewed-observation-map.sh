#!/usr/bin/env bash
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
cd "$root"

output_dir="${1:-artifacts/testule}"
max_bytes="1048576"
stream="$output_dir/reviewed-go-observation-map.json"
revision="$output_dir/reviewed-go-map-subject-revision.txt"

targets=(
  "repoctl/internal/apply|TestPreflightRepositoryArtifactAuditedChecksEveryTargetBeforeMutation"
  "repoctl/cmd/repoctl|TestStatusOutputMatchesGoldenContract"
  "repoctl/internal/apply|TestExecuteSynchronizesBehindMirrorUsingLocalGitRepos"
  "repoctl/internal/plan|TestReconcileRejectsUnsupportedTopologyWithoutPartialPlan"
  "repoctl/cmd/repoctl|TestApplyHonorsParallelLimit"
  "repoctl/internal/git|TestEnsureMirrorRejectsSymlinkEscape"
)

target_pattern='^(TestPreflightRepositoryArtifactAuditedChecksEveryTargetBeforeMutation|TestStatusOutputMatchesGoldenContract|TestExecuteSynchronizesBehindMirrorUsingLocalGitRepos|TestReconcileRejectsUnsupportedTopologyWithoutPartialPlan|TestApplyHonorsParallelLimit|TestEnsureMirrorRejectsSymlinkEscape)$'

mkdir -p "$output_dir"
umask 077
rm -f "$stream" "$revision"

set +e
go test -race -count=1 -json \
  ./internal/apply ./cmd/repoctl ./internal/plan ./internal/git \
  -run "$target_pattern" \
  | python3 ./scripts/ci/capture-json-stream.py "$stream" "$max_bytes"
pipeline_status=("${PIPESTATUS[@]}")
set -e

if [[ "${pipeline_status[0]}" -ne 0 ]]; then
  printf 'reviewed native Go observation map failed; refusing source material\n' >&2
  exit 1
fi
if [[ "${pipeline_status[1]}" -ne 0 ]]; then
  printf 'reviewed native Go JSON capture failed; refusing partial/oversized source\n' >&2
  exit 1
fi

for entry in "${targets[@]}"; do
  package="${entry%%|*}"
  target="${entry#*|}"
  python3 ./scripts/ci/verify-go-test-observation.py "$stream" "$package" "$target"
done

git rev-parse HEAD > "$revision"
printf 'reviewed native observation map: %d targets @ %s\n' "${#targets[@]}" "$(cat "$revision")"
