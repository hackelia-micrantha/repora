#!/usr/bin/env python3
from __future__ import annotations

import fnmatch
import json
import sys
from pathlib import Path

EXPECTED_TIERS = [
    "canonical",
    "implementation",
    "generated",
    "experimental",
    "archived",
    "external",
]


def fail(message: str) -> None:
    raise SystemExit(f"trust-policy: {message}")


def parse_list(lines: list[str], start: int, indent: int) -> tuple[list[str], int]:
    values: list[str] = []
    i = start
    while i < len(lines):
        line = lines[i]
        stripped = line.strip()
        current = len(line) - len(line.lstrip())
        if stripped and current < indent:
            break
        if current == indent and stripped.startswith("- "):
            values.append(stripped[2:].strip())
        i += 1
    return values, i


def parse_policy(path: Path) -> dict:
    lines = path.read_text(encoding="utf-8").splitlines()
    try:
        trust_index = next(i for i, line in enumerate(lines) if line == "trust:")
        classes_index = next(i for i, line in enumerate(lines[trust_index + 1 :], trust_index + 1) if line == "classes:")
    except StopIteration:
        fail("router must contain trust and classes sections")

    section = lines[trust_index + 1 : classes_index]
    policy: dict[str, object] = {"rules": []}
    i = 0
    while i < len(section):
        stripped = section[i].strip()
        if stripped in {"tiers:", "default_include:", "explicit_include_required:"}:
            key = stripped[:-1]
            values, i = parse_list(section, i + 1, 4)
            policy[key] = values
            continue
        if stripped.startswith("unclassified:"):
            policy["unclassified"] = stripped.split(":", 1)[1].strip()
        if stripped == "rules:":
            i += 1
            while i < len(section):
                line = section[i]
                stripped_rule = line.strip()
                if stripped_rule.startswith("- tier:"):
                    tier = stripped_rule.split(":", 1)[1].strip()
                    i += 1
                    if i >= len(section) or section[i].strip() != "paths:":
                        fail(f"trust rule {tier!r} must define paths")
                    paths, i = parse_list(section, i + 1, 8)
                    policy["rules"].append({"tier": tier, "paths": paths})
                    continue
                i += 1
            break
        i += 1
    return policy


def specificity(pattern: str) -> tuple[int, int]:
    literal = pattern.replace("**", "").replace("*", "").replace("?", "")
    return (len(literal), len(pattern))


def classify(policy: dict, candidate: str) -> str:
    matches: list[tuple[tuple[int, int], int, str]] = []
    for rule_index, rule in enumerate(policy["rules"]):
        for pattern in rule["paths"]:
            if fnmatch.fnmatchcase(candidate, pattern):
                matches.append((specificity(pattern), -rule_index, rule["tier"]))
    if not matches:
        return "unclassified"
    matches.sort(reverse=True)
    return matches[0][2]


def main() -> None:
    if len(sys.argv) != 3:
        fail("usage: trust-policy.py <router.yaml> <fixtures.json>")
    policy = parse_policy(Path(sys.argv[1]))
    fixtures = json.loads(Path(sys.argv[2]).read_text(encoding="utf-8"))

    if policy.get("tiers") != EXPECTED_TIERS:
        fail(f"tiers: got {policy.get('tiers')}, want {EXPECTED_TIERS}")
    if policy.get("default_include") != ["canonical", "implementation"]:
        fail("default_include must contain only canonical and implementation")
    if policy.get("explicit_include_required") != ["generated", "experimental", "archived", "external"]:
        fail("explicit_include_required must contain all lower-trust tiers")
    if policy.get("unclassified") != "exclude":
        fail("unclassified content must fail closed")

    seen_patterns: set[str] = set()
    for rule in policy["rules"]:
        if rule["tier"] not in EXPECTED_TIERS:
            fail(f"unknown tier {rule['tier']!r}")
        if not rule["paths"]:
            fail(f"tier {rule['tier']!r} has no paths")
        for pattern in rule["paths"]:
            if pattern in seen_patterns:
                fail(f"duplicate trust pattern {pattern!r}")
            seen_patterns.add(pattern)

    if fixtures.get("version") != 1 or fixtures.get("kind") != "document-trust-tests":
        fail("unsupported fixture contract")

    default_tiers = set(policy["default_include"])
    for case in fixtures.get("cases", []):
        tier = classify(policy, case["path"])
        if tier != case["expect_tier"]:
            fail(f"{case['name']}: tier got {tier!r}, want {case['expect_tier']!r}")
        default_eligible = tier in default_tiers
        if default_eligible != case["default_eligible"]:
            fail(f"{case['name']}: default eligibility mismatch")
        explicit = set(case.get("explicit_include", []))
        explicit_eligible = default_eligible or tier in explicit
        if "explicit_eligible" in case and explicit_eligible != case["explicit_eligible"]:
            fail(f"{case['name']}: explicit eligibility mismatch")
        print(f"ok: {case['name']}")

    print(f"validated {len(fixtures.get('cases', []))} document trust fixtures")


if __name__ == "__main__":
    main()
