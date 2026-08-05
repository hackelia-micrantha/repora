#!/usr/bin/env python3
"""Validate deterministic and security-sensitive context receipt invariants."""

from __future__ import annotations

import hashlib
import json
import re
import sys
from pathlib import Path, PurePosixPath
from typing import Any

SHA256 = re.compile(r"^[0-9a-f]{64}$")
GIT_SHA = re.compile(r"^[0-9a-f]{40}$")
SENSITIVE = re.compile(
    r"(?i)(authorization\s*:|bearer\s+[a-z0-9._~-]+|password\s*[=:]|token\s*[=:]|api[_-]?key\s*[=:]|-----begin .*private key-----)"
)
CREDENTIAL_URL = re.compile(r"https?://[^/\s:@]+:[^/\s@]+@")
TRUST_TIERS = {
    "canonical",
    "implementation",
    "generated",
    "experimental",
    "archived",
    "external",
}
EXCLUSION_REASONS = {
    "route_exclude",
    "trust_exclude",
    "budget_exceeded",
    "unclassified",
    "duplicate",
}
TOP_LEVEL_FIELDS = {
    "version",
    "kind",
    "query",
    "repository",
    "routing",
    "budget",
    "selected",
    "excluded",
}


def fail(message: str) -> None:
    raise ValueError(message)


def require_object(value: Any, field: str, keys: set[str]) -> dict[str, Any]:
    if not isinstance(value, dict):
        fail(f"{field} must be an object")
    if set(value) != keys:
        fail(f"{field} has missing or unknown fields")
    return value


def require_array(value: Any, field: str) -> list[Any]:
    if not isinstance(value, list):
        fail(f"{field} must be an array")
    return value


def repository_path(value: Any, field: str) -> str:
    if not isinstance(value, str):
        fail(f"{field} must be a string")
    path = PurePosixPath(value)
    if not value or path.is_absolute() or ".." in path.parts or "\\" in value:
        fail(f"{field} must be a safe repository-relative POSIX path: {value!r}")
    return value


def require_sha(value: Any, field: str) -> str:
    if not isinstance(value, str) or not SHA256.fullmatch(value):
        fail(f"{field} must be a lowercase SHA-256 digest")
    return value


def require_non_negative_int(value: Any, field: str) -> int:
    if not isinstance(value, int) or isinstance(value, bool) or value < 0:
        fail(f"{field} must be a non-negative integer")
    return value


def canonical_bytes(receipt: dict[str, Any]) -> bytes:
    return (
        json.dumps(receipt, indent=2, sort_keys=True, ensure_ascii=False) + "\n"
    ).encode("utf-8")


def validate(receipt: Any, raw: str | None = None) -> str:
    receipt = require_object(receipt, "receipt", TOP_LEVEL_FIELDS)
    if receipt["version"] != 1 or receipt["kind"] != "context-receipt":
        fail("unsupported receipt contract")

    query = receipt["query"]
    if not isinstance(query, str) or not query.strip():
        fail("query must be non-empty")
    if len(query) > 4096:
        fail("query exceeds the v1 bound")
    if query != " ".join(query.split()):
        fail("query whitespace is not normalized")

    repository = require_object(
        receipt["repository"], "repository", {"revision", "dirty"}
    )
    if not isinstance(repository["revision"], str) or not GIT_SHA.fullmatch(
        repository["revision"]
    ):
        fail("repository.revision must be a full lowercase Git SHA")
    if not isinstance(repository["dirty"], bool):
        fail("repository.dirty must be boolean")

    routing = require_object(
        receipt["routing"],
        "routing",
        {"policy_version", "policy_sha256", "routes", "fallback", "manifests"},
    )
    if (
        not isinstance(routing["policy_version"], int)
        or isinstance(routing["policy_version"], bool)
        or routing["policy_version"] < 1
    ):
        fail("routing.policy_version must be a positive integer")
    require_sha(routing["policy_sha256"], "routing.policy_sha256")

    routes = require_array(routing["routes"], "routing.routes")
    route_ids: list[str] = []
    for index, route_value in enumerate(routes):
        route = require_object(
            route_value, f"routing.routes[{index}]", {"id", "match_terms"}
        )
        route_id = route["id"]
        if not isinstance(route_id, str) or not route_id.strip():
            fail(f"routing.routes[{index}].id must be non-empty")
        route_ids.append(route_id)
        terms = require_array(
            route["match_terms"], f"routing.routes[{index}].match_terms"
        )
        if any(
            not isinstance(term, str)
            or not term.strip()
            or term != " ".join(term.split())
            for term in terms
        ):
            fail(
                f"routing.routes[{index}].match_terms must contain normalized non-empty strings"
            )
        if terms != sorted(set(terms)):
            fail(f"routing.routes[{index}].match_terms must be unique and sorted")
    if len(route_ids) != len(set(route_ids)):
        fail("routing route IDs must be unique")

    fallback = routing["fallback"]
    if fallback is not None and (
        not isinstance(fallback, str) or not fallback.strip()
    ):
        fail("routing.fallback must be null or a non-empty string")
    if bool(routes) == bool(fallback):
        fail("receipt must record either selected routes or one fallback, but not both")

    manifests = require_array(routing["manifests"], "routing.manifests")
    manifest_paths = [
        repository_path(item, "routing.manifests[]") for item in manifests
    ]
    if manifest_paths != sorted(set(manifest_paths)):
        fail("routing manifests must be unique and sorted")

    budget = require_object(
        receipt["budget"],
        "budget",
        {
            "max_files",
            "max_bytes",
            "max_tokens_hint",
            "selected_files",
            "selected_bytes",
            "estimated_tokens",
        },
    )
    for field in budget:
        require_non_negative_int(budget[field], f"budget.{field}")

    selected = require_array(receipt["selected"], "selected")
    selected_paths: list[str] = []
    selected_bytes = 0
    for item_index, item_value in enumerate(selected):
        item = require_object(
            item_value,
            f"selected[{item_index}]",
            {"path", "trust_tier", "reason", "bytes", "content_sha256", "snippets"},
        )
        path = repository_path(item["path"], f"selected[{item_index}].path")
        selected_paths.append(path)
        if item["trust_tier"] not in TRUST_TIERS:
            fail(f"unknown trust tier for {path!r}")
        if not isinstance(item["reason"], str) or not item["reason"].strip():
            fail(f"selected reason must be non-empty for {path!r}")
        item_bytes = require_non_negative_int(
            item["bytes"], f"selected[{path}].bytes"
        )
        selected_bytes += item_bytes
        require_sha(item["content_sha256"], f"selected[{path}].content_sha256")

        snippets = require_array(item["snippets"], f"selected[{path}].snippets")
        indices: list[int] = []
        snippet_bytes = 0
        for snippet_index, snippet_value in enumerate(snippets):
            snippet = require_object(
                snippet_value,
                f"selected[{path}].snippets[{snippet_index}]",
                {"index", "sha256", "bytes"},
            )
            index = require_non_negative_int(
                snippet["index"], f"selected[{path}].snippet.index"
            )
            indices.append(index)
            require_sha(snippet["sha256"], f"selected[{path}].snippet.sha256")
            size = require_non_negative_int(
                snippet["bytes"], f"selected[{path}].snippet.bytes"
            )
            if size > 4096:
                fail(f"snippet size is outside the v1 bound for {path!r}")
            snippet_bytes += size
        if indices != sorted(set(indices)):
            fail(f"snippet indices must be unique and sorted for {path!r}")
        if snippet_bytes > item_bytes:
            fail(f"snippet bytes exceed selected input bytes for {path!r}")

    if selected_paths != sorted(selected_paths) or len(selected_paths) != len(
        set(selected_paths)
    ):
        fail("selected inputs must be unique and path-sorted")
    if budget["selected_files"] != len(selected):
        fail("budget.selected_files does not match selected input count")
    if budget["selected_bytes"] != selected_bytes:
        fail("budget.selected_bytes does not match selected input bytes")
    if budget["selected_files"] > budget["max_files"]:
        fail("selected file count exceeds max_files")
    if budget["selected_bytes"] > budget["max_bytes"]:
        fail("selected bytes exceed max_bytes")
    if budget["estimated_tokens"] > budget["max_tokens_hint"]:
        fail("estimated tokens exceed max_tokens_hint")

    excluded = require_array(receipt["excluded"], "excluded")
    excluded_keys: list[tuple[str, str]] = []
    for item_index, item_value in enumerate(excluded):
        if not isinstance(item_value, dict):
            fail(f"excluded[{item_index}] must be an object")
        allowed = {"path", "reason", "trust_tier"}
        if not {"path", "reason"}.issubset(item_value) or not set(
            item_value
        ).issubset(allowed):
            fail(f"excluded[{item_index}] has missing or unknown fields")
        path = repository_path(item_value["path"], f"excluded[{item_index}].path")
        reason = item_value["reason"]
        if reason not in EXCLUSION_REASONS:
            fail(f"unknown exclusion reason for {path!r}")
        tier = item_value.get("trust_tier")
        if tier is not None and tier not in TRUST_TIERS:
            fail(f"unknown excluded trust tier for {path!r}")
        excluded_keys.append((path, reason))
    if excluded_keys != sorted(excluded_keys) or len(excluded_keys) != len(
        set(excluded_keys)
    ):
        fail("excluded inputs must be unique and sorted by path and reason")

    serialized = json.dumps(receipt, ensure_ascii=False)
    if SENSITIVE.search(serialized):
        fail("receipt contains credential-like material")
    if CREDENTIAL_URL.search(serialized):
        fail("receipt contains a credential-bearing URL")

    canonical = canonical_bytes(receipt)
    if raw is not None and raw.encode("utf-8") != canonical:
        fail("receipt is not in canonical deterministic JSON form")
    return hashlib.sha256(canonical).hexdigest()


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit("context-receipt: usage: context-receipt.py <receipt.json>")

    path = Path(sys.argv[1])
    try:
        raw = path.read_text(encoding="utf-8")
        receipt = json.loads(raw)
        digest = validate(receipt, raw)
    except (OSError, json.JSONDecodeError, ValueError) as exc:
        raise SystemExit(f"context-receipt: {exc}") from exc

    print(f"validated context receipt {path} sha256={digest}")


if __name__ == "__main__":
    main()
