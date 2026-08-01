#!/usr/bin/env bash
set -euo pipefail

count="${REPEAT_COUNT:-10}"
if ! [[ "$count" =~ ^[1-9][0-9]*$ ]]; then
  echo "REPEAT_COUNT must be a positive integer, got: $count" >&2
  exit 2
fi

if [[ "$#" -eq 0 ]]; then
  set -- ./...
fi

printf 'repeated test command: go test -race -count=1 -short %q' "$1"
for package in "${@:2}"; do
  printf ' %q' "$package"
done
printf '\npackages: %s\niterations: %s\n' "$*" "$count"

for ((iteration = 1; iteration <= count; iteration++)); do
  echo "iteration ${iteration}/${count}: go test -race -count=1 -short $*"
  if ! go test -race -count=1 -short "$@"; then
    echo "repeated test failure: iteration=${iteration} count=${count} packages=$*" >&2
    exit 1
  fi
done
