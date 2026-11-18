#!/usr/bin/env bash
set -euo pipefail

# Acceptance only: fixed public GitLab canonical -> GitHub mirror; never push.
# Run only in a clean ephemeral CI worker, without repository write credentials.
if [[ "$#" -ne 2 ]]; then
  printf 'usage: public-mirror-smoke.sh LINUX_RELEASE_ARCHIVE EVIDENCE_DIR\n' >&2
  exit 2
fi
archive="$1"
evidence_dir="$2"
if [[ ! -f "$archive" ]]; then
  printf 'missing preverified Linux archive: %s\n' "$archive" >&2
  exit 1
fi

mkdir -p "$evidence_dir"
evidence_dir="$(cd "$evidence_dir" && pwd)"
work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT
mkdir -p "$work_dir/home" "$work_dir/cache"

# Suppress interactive credentials, SSH, global Git config and inherited tokens.
export HOME="$work_dir/home"
export REPORA_CACHE_DIR="$work_dir/cache"
export GIT_CONFIG_NOSYSTEM=1
export GIT_CONFIG_GLOBAL=/dev/null
export GIT_TERMINAL_PROMPT=0
export GIT_ASKPASS=/bin/false
export SSH_ASKPASS=/bin/false
export GIT_ALLOW_PROTOCOL=https
unset GH_TOKEN GITHUB_TOKEN GITLAB_TOKEN GITLAB_PRIVATE_TOKEN GIT_ASKPASS_REQUIRE

# Fixed non-secret public pair; no private topology or configurable targets.
cat >"$work_dir/public-smoke.yaml" <<'YAML'
repos:
  - id: gitlab-shell-public
    uid: public.gitlab-shell
    canonical:
      provider: gitlab
      path: gitlab-org/gitlab-shell
    mirrors:
      - provider: github
        path: gitlabhq/gitlab-shell
    mode: mirror
YAML

tar -xzf "$archive" -C "$work_dir"
shopt -s nullglob
binaries=("$work_dir"/repoctl_*/repoctl)
if [[ "${#binaries[@]}" -ne 1 || ! -x "${binaries[0]}" ]]; then
  printf 'archive must have exactly one executable repoctl at package root\n' >&2
  exit 1
fi
binary="${binaries[0]}"

"$binary" --version >"$evidence_dir/version.txt"
: "${VERSION:?VERSION must match the packaged candidate}"
: "${COMMIT:?COMMIT must match the packaged candidate}"
if [[ "$(cat "$evidence_dir/version.txt")" != "repoctl $VERSION ($COMMIT)" ]]; then
  printf 'candidate archive version/commit mismatch\n' >&2
  exit 1
fi
sha256sum "$archive" >"$evidence_dir/archive.sha256"
sha256sum "$work_dir/public-smoke.yaml" >"$evidence_dir/topology.sha256"

run_capture() {
  local label="$1"
  shift
  local code
  set +e
  "$@" >"$evidence_dir/$label.json" 2>"$evidence_dir/$label.stderr"
  code=$?
  set -e
  printf '%s\t%s\n' "$label" "$code" >>"$evidence_dir/exit-codes.tsv"
  return "$code"
}

printf 'command\texit_code\n' >"$evidence_dir/exit-codes.tsv"
if ! run_capture status "$binary" status -f "$work_dir/public-smoke.yaml" --json; then
  printf 'public mirror status failed; no plan/apply attempted\n' >&2
  exit 1
fi
if ! run_capture plan "$binary" plan -f "$work_dir/public-smoke.yaml" --artifact; then
  printf 'public mirror plan failed; no apply attempted\n' >&2
  exit 1
fi
# Consume exact full plan artifact; never truncate or reconstruct the plan.
if ! run_capture dry-run "$binary" apply -f "$work_dir/public-smoke.yaml" --plan-file "$evidence_dir/plan.json" --dry-run --json; then
  printf 'public mirror dry-run failed; no remote write attempted\n' >&2
  exit 1
fi

printf 'public GitLab/GitHub status, plan and exact-plan dry-run passed with exit 0; no remote write\n'
sha256sum "$evidence_dir"/*.json >"$evidence_dir/output-sha256.txt"
