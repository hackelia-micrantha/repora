#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root"

: "${VERSION:?VERSION is required}"
: "${COMMIT:?COMMIT is required}"

release_version="${VERSION#v}"
dist="$root/dist"
expected=(
  "repoctl_${release_version}_linux_amd64.tar.gz"
  "repoctl_${release_version}_darwin_amd64.tar.gz"
  "repoctl_${release_version}_darwin_arm64.tar.gz"
  "repoctl_${release_version}_windows_amd64.zip"
)

for file in "${expected[@]}"; do
  if [[ ! -f "$dist/$file" ]]; then
    printf 'missing release artifact %s\n' "$file" >&2
    exit 1
  fi
done
if [[ ! -f "$dist/checksums.txt" ]]; then
  printf 'missing checksums.txt\n' >&2
  exit 1
fi
if [[ "$(wc -l <"$dist/checksums.txt")" -ne "${#expected[@]}" ]]; then
  printf 'checksums.txt must contain exactly %d archive entries\n' "${#expected[@]}" >&2
  exit 1
fi
(
  cd "$dist"
  sha256sum -c checksums.txt
)

tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT

linux_package="repoctl_${release_version}_linux_amd64"
tar -xzf "$dist/$linux_package.tar.gz" -C "$tmp_root"
version_output="$($tmp_root/$linux_package/repoctl --version)"
expected_version="repoctl $VERSION ($COMMIT)"
if [[ "$version_output" != "$expected_version" ]]; then
  printf 'packaged version = %q, want %q\n' "$version_output" "$expected_version" >&2
  exit 1
fi
bash ./scripts/ci/cli-smoke.sh "$tmp_root/$linux_package/repoctl"

for file in "${expected[@]}"; do
  package="${file%.tar.gz}"
  package="${package%.zip}"
  if [[ "$file" == *.zip ]]; then
    listing="$(unzip -Z1 "$dist/$file")"
  else
    listing="$(tar -tzf "$dist/$file")"
  fi
  for member in "$package/repoctl" "$package/LICENSE" "$package/README.md"; do
    if [[ "$file" == *.zip && "$member" == "$package/repoctl" ]]; then
      member="$package/repoctl.exe"
    fi
    if ! grep -Fxq "$member" <<<"$listing"; then
      printf '%s is missing %s\n' "$file" "$member" >&2
      exit 1
    fi
  done
done

printf 'verified %s release packages\n' "$VERSION"
