#!/usr/bin/env python3
"""Validate Repora assessment examples and templates without external dependencies."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path

FINDING_TYPES = {"question", "finding", "recommendation", "tradeoff", "risk", "gap", "overlap", "drift"}
SEVERITIES = {"critical", "high", "medium", "low", "informational"}
STATUSES = {"open", "accepted", "deferred", "implemented", "rejected"}
EVIDENCE_CATEGORIES = {
    "architecture", "security", "testing", "devops", "observability", "backend",
    "mobile", "frontend", "platform", "ai", "leadership", "mentorship",
}
EVIDENCE_STRENGTH = {"strong", "moderate", "weak", "unsupported"}
SCORE_DIMENSIONS = {"architecture", "security", "testing", "delivery", "operations", "maintainability", "documentation"}
REFERENCE_TYPES = {"issue", "pull_request", "commit", "file", "url"}
ID_RE = re.compile(r"^[a-z0-9][a-z0-9._-]*$")
COMMIT_RE = re.compile(r"^[0-9a-fA-F]{7,64}$")


def fail(path: Path, message: str) -> None:
    raise SystemExit(f"assessment-contract: {path}: {message}")


def require(condition: bool, path: Path, message: str) -> None:
    if not condition:
        fail(path, message)


def load(path: Path) -> dict:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        fail(path, f"cannot parse JSON: {exc}")
    require(isinstance(value, dict), path, "document must be an object")
    return value


def validate_reference(path: Path, ref: object, context: str) -> None:
    require(isinstance(ref, dict), path, f"{context} reference must be an object")
    require(ref.get("type") in REFERENCE_TYPES, path, f"{context} has invalid reference type")
    require(isinstance(ref.get("value"), str) and ref["value"].strip(), path, f"{context} reference value is required")


def validate_snapshot(path: Path, snapshot: object) -> None:
    require(isinstance(snapshot, dict), path, "snapshot must be an object")
    require(snapshot.get("kind") == "repora.repository-snapshot", path, "invalid snapshot kind")
    require(snapshot.get("version") == 1, path, "invalid snapshot version")
    repository = snapshot.get("repository")
    require(isinstance(repository, dict), path, "snapshot repository must be an object")
    require(isinstance(repository.get("full_name"), str) and repository["full_name"].strip(), path, "repository full_name is required")
    revision = snapshot.get("revision")
    require(isinstance(revision, dict), path, "snapshot revision must be an object")
    require(isinstance(revision.get("commit"), str) and COMMIT_RE.fullmatch(revision["commit"]), path, "snapshot commit must be a 7-64 character hex revision")
    require(isinstance(revision.get("dirty"), bool), path, "snapshot dirty must be boolean")
    require(isinstance(snapshot.get("captured_at"), str) and snapshot["captured_at"].strip(), path, "captured_at is required")


def validate_assessment(path: Path) -> None:
    doc = load(path)
    require(doc.get("kind") == "repora.repository-assessment", path, "invalid assessment kind")
    require(doc.get("version") == 1, path, "invalid assessment version")
    require(isinstance(doc.get("id"), str) and ID_RE.fullmatch(doc["id"]), path, "invalid assessment id")
    require(isinstance(doc.get("title"), str) and doc["title"].strip(), path, "title is required")
    require(isinstance(doc.get("summary"), str) and doc["summary"].strip(), path, "summary is required")
    require(isinstance(doc.get("scope"), list) and doc["scope"], path, "scope must be non-empty")
    validate_snapshot(path, doc.get("snapshot"))

    evidence = doc.get("evidence")
    findings = doc.get("findings")
    scorecard = doc.get("scorecard")
    require(isinstance(evidence, list), path, "evidence must be an array")
    require(isinstance(findings, list), path, "findings must be an array")
    require(isinstance(scorecard, dict), path, "scorecard must be an object")

    evidence_ids: set[str] = set()
    for item in evidence:
        require(isinstance(item, dict), path, "evidence item must be an object")
        require(item.get("kind") == "repora.evidence" and item.get("version") == 1, path, "invalid evidence kind/version")
        evidence_id = item.get("id")
        require(isinstance(evidence_id, str) and ID_RE.fullmatch(evidence_id), path, "invalid evidence id")
        require(evidence_id not in evidence_ids, path, f"duplicate evidence id {evidence_id!r}")
        evidence_ids.add(evidence_id)
        require(item.get("category") in EVIDENCE_CATEGORIES, path, f"invalid evidence category for {evidence_id}")
        require(item.get("strength") in EVIDENCE_STRENGTH, path, f"invalid evidence strength for {evidence_id}")
        require(isinstance(item.get("claim"), str) and item["claim"].strip(), path, f"evidence {evidence_id} claim is required")
        require(isinstance(item.get("rationale"), str) and item["rationale"].strip(), path, f"evidence {evidence_id} rationale is required")
        refs = item.get("references")
        require(isinstance(refs, list), path, f"evidence {evidence_id} references must be an array")
        if item["strength"] != "unsupported":
            require(bool(refs), path, f"evidence {evidence_id} with strength {item['strength']} requires a reference")
        for ref in refs:
            validate_reference(path, ref, f"evidence {evidence_id}")

    finding_ids: set[str] = set()
    for item in findings:
        require(isinstance(item, dict), path, "finding item must be an object")
        require(item.get("kind") == "repora.finding" and item.get("version") == 1, path, "invalid finding kind/version")
        finding_id = item.get("id")
        require(isinstance(finding_id, str) and ID_RE.fullmatch(finding_id), path, "invalid finding id")
        require(finding_id not in finding_ids, path, f"duplicate finding id {finding_id!r}")
        finding_ids.add(finding_id)
        require(item.get("type") in FINDING_TYPES, path, f"invalid finding type for {finding_id}")
        require(item.get("severity") in SEVERITIES, path, f"invalid severity for {finding_id}")
        require(item.get("status") in STATUSES, path, f"invalid status for {finding_id}")
        require(isinstance(item.get("title"), str) and item["title"].strip(), path, f"finding {finding_id} title is required")
        require(isinstance(item.get("description"), str) and item["description"].strip(), path, f"finding {finding_id} description is required")
        refs = item.get("references")
        require(isinstance(refs, list), path, f"finding {finding_id} references must be an array")
        for ref in refs:
            validate_reference(path, ref, f"finding {finding_id}")
        linked = item.get("evidence_ids")
        require(isinstance(linked, list) and len(linked) == len(set(linked)), path, f"finding {finding_id} evidence_ids must be unique")
        unknown = sorted(set(linked) - evidence_ids)
        require(not unknown, path, f"finding {finding_id} references unknown evidence ids {unknown}")

    require(scorecard.get("kind") == "repora.scorecard" and scorecard.get("version") == 1, path, "invalid scorecard kind/version")
    dimensions = scorecard.get("dimensions")
    require(isinstance(dimensions, list) and dimensions, path, "scorecard dimensions must be non-empty")
    dimension_names: set[str] = set()
    for dimension in dimensions:
        require(isinstance(dimension, dict), path, "scorecard dimension must be an object")
        name = dimension.get("name")
        require(name in SCORE_DIMENSIONS, path, f"invalid scorecard dimension {name!r}")
        require(name not in dimension_names, path, f"duplicate scorecard dimension {name!r}")
        dimension_names.add(name)
        score = dimension.get("score")
        require(isinstance(score, int) and not isinstance(score, bool) and 0 <= score <= 5, path, f"score for {name} must be integer 0-5")
        require(isinstance(dimension.get("rationale"), str) and dimension["rationale"].strip(), path, f"scorecard rationale for {name} is required")
        linked = dimension.get("evidence_ids")
        require(isinstance(linked, list) and len(linked) == len(set(linked)), path, f"scorecard evidence_ids for {name} must be unique")
        unknown = sorted(set(linked) - evidence_ids)
        require(not unknown, path, f"scorecard {name} references unknown evidence ids {unknown}")

    print(f"ok: {path}")


def main() -> None:
    if len(sys.argv) < 2:
        raise SystemExit("usage: assessment-contract.py <assessment.json> [assessment.json ...]")
    for arg in sys.argv[1:]:
        validate_assessment(Path(arg))
    print(f"validated {len(sys.argv) - 1} assessment documents")


if __name__ == "__main__":
    main()
