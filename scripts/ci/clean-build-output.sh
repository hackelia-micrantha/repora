#!/usr/bin/env bash
# Remove only known repoctl binaries generated in this Git worktree.
# Never sweep release or validation evidence (dist/, artifacts/), or shared Go caches.
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "$0")/../.." && pwd -P)"
bin_dir="$repo_root/bin"
if [[ -L "$bin_dir" ]]; then
  echo "refusing symlinked bin directory: $bin_dir" >&2
  exit 2
fi
if [[ -e "$bin_dir" && ! -d "$bin_dir" ]]; then
  echo "refusing non-directory build output: $bin_dir" >&2
  exit 2
fi

for output in \
  "$repo_root/repoctl" \
  "$bin_dir/repoctl" \
  "$bin_dir/repoctl-linux-amd64" \
  "$bin_dir/repoctl-windows-amd64.exe" \
  "$bin_dir/repoctl-darwin-amd64" \
  "$bin_dir/repoctl-darwin-arm64"; do
  if [[ -L "$output" ]]; then
    echo "refusing symlinked build output: $output" >&2
    exit 2
  fi
  if [[ -e "$output" && ! -f "$output" ]]; then
    echo "refusing non-file build output: $output" >&2
    exit 2
  fi
done

for output in \
  "$repo_root/repoctl" \
  "$bin_dir/repoctl" \
  "$bin_dir/repoctl-linux-amd64" \
  "$bin_dir/repoctl-windows-amd64.exe" \
  "$bin_dir/repoctl-darwin-amd64" \
  "$bin_dir/repoctl-darwin-arm64"; do
  if [[ -f "$output" ]]; then
    rm -- "$output"
  fi
done
