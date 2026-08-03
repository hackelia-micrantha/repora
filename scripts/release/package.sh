#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root"

: "${VERSION:?VERSION is required, for example v0.1.0}"
: "${COMMIT:?COMMIT is required}"
: "${SOURCE_DATE_EPOCH:?SOURCE_DATE_EPOCH is required}"

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$ ]]; then
  printf 'VERSION must be a v-prefixed semantic version, got %q\n' "$VERSION" >&2
  exit 2
fi
if [[ ! "$COMMIT" =~ ^[0-9a-f]{7,64}$ ]]; then
  printf 'COMMIT must be a lowercase hexadecimal Git object ID, got %q\n' "$COMMIT" >&2
  exit 2
fi
if [[ ! "$SOURCE_DATE_EPOCH" =~ ^[0-9]+$ ]]; then
  printf 'SOURCE_DATE_EPOCH must be an integer, got %q\n' "$SOURCE_DATE_EPOCH" >&2
  exit 2
fi

release_version="${VERSION#v}"
dist="$root/dist"
tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT

rm -rf "$dist"
mkdir -p "$dist"

targets=(
  "linux amd64"
  "darwin amd64"
  "darwin arm64"
  "windows amd64"
)

for target in "${targets[@]}"; do
  read -r goos goarch <<<"$target"
  package="repoctl_${release_version}_${goos}_${goarch}"
  stage="$tmp_root/$package"
  binary="repoctl"
  archive="$dist/$package.tar.gz"
  if [[ "$goos" == "windows" ]]; then
    binary="repoctl.exe"
    archive="$dist/$package.zip"
  fi

  mkdir -p "$stage"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -buildvcs=false \
      -ldflags "-s -w -X main.version=$VERSION -X main.commit=$COMMIT" \
      -o "$stage/$binary" ./cmd/repoctl
  cp LICENSE README.md "$stage/"
  find "$stage" -exec touch --date="@$SOURCE_DATE_EPOCH" {} +

  if [[ "$goos" == "windows" ]]; then
    (
      cd "$tmp_root"
      zip -X -q -r "$archive" "$package"
    )
  else
    tar --sort=name --owner=0 --group=0 --numeric-owner \
      --mtime="@$SOURCE_DATE_EPOCH" -C "$tmp_root" -cf - "$package" | \
      gzip -n >"$archive"
  fi
done

(
  cd "$dist"
  sha256sum ./*.tar.gz ./*.zip | LC_ALL=C sort -k2 >checksums.txt
)

printf 'packaged %s (%s) in %s\n' "$VERSION" "$COMMIT" "$dist"
