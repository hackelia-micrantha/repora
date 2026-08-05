#!/usr/bin/env python3
"""Validate that configured route manifests resolve inside the repository."""

from __future__ import annotations

import argparse
from pathlib import Path


def parse_manifest_paths(router_path: Path) -> list[str]:
    paths: list[str] = []
    in_manifests = False
    for raw_line in router_path.read_text(encoding="utf-8").splitlines():
        if raw_line == "manifests:":
            in_manifests = True
            continue
        if not in_manifests:
            continue
        if raw_line.startswith("  - "):
            value = raw_line[4:].strip()
            if not value:
                raise ValueError("manifest path must not be empty")
            paths.append(value)
            continue
        if raw_line and not raw_line.startswith(" "):
            break
    return paths


def validate_manifest_paths(router_path: Path) -> list[Path]:
    router_path = router_path.resolve(strict=True)
    repository_root = router_path.parent.parent.resolve(strict=True)
    validated: list[Path] = []

    for configured in parse_manifest_paths(router_path):
        candidate = Path(configured)
        if candidate.is_absolute() or ".." in candidate.parts:
            raise ValueError(f"unsafe manifest path {configured!r}")

        resolved = (repository_root / candidate).resolve(strict=True)
        try:
            resolved.relative_to(repository_root)
        except ValueError as exc:
            raise ValueError(
                f"manifest path {configured!r} resolves outside repository"
            ) from exc
        if not resolved.is_file():
            raise ValueError(f"manifest path {configured!r} is not a regular file")
        validated.append(resolved)

    return validated


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("router", type=Path)
    args = parser.parse_args()

    validated = validate_manifest_paths(args.router)
    print(f"validated {len(validated)} manifest paths inside repository")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
