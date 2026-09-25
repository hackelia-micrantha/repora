#!/usr/bin/env python3
import json
import sys
from pathlib import Path

MAX_INPUT_BYTES = 1024 * 1024
TERMINAL_ACTIONS = {"pass", "fail", "skip"}


def main(argv):
    if len(argv) != 4:
        print(f"usage: {argv[0]} <go-test-json> <package> <target>", file=sys.stderr)
        return 2

    path = Path(argv[1])
    package = argv[2]
    target = argv[3]

    try:
        size = path.stat().st_size
    except OSError as exc:
        print(f"cannot stat observation stream {path}: {exc}", file=sys.stderr)
        return 1
    if size > MAX_INPUT_BYTES:
        print(f"observation stream exceeds {MAX_INPUT_BYTES} bytes", file=sys.stderr)
        return 1

    target_actions = []
    terminal_actions = []
    try:
        with path.open("r", encoding="utf-8") as source:
            for line_number, raw in enumerate(source, 1):
                if not raw.strip():
                    continue
                try:
                    event = json.loads(raw)
                except json.JSONDecodeError as exc:
                    print(f"invalid JSON at line {line_number}: {exc}", file=sys.stderr)
                    return 1
                if not isinstance(event, dict):
                    print(f"non-object JSON event at line {line_number}", file=sys.stderr)
                    return 1
                if event.get("Package") != package or event.get("Test") != target:
                    continue
                action = event.get("Action")
                if isinstance(action, str):
                    target_actions.append(action)
                if action in TERMINAL_ACTIONS:
                    terminal_actions.append(str(action))
    except (OSError, UnicodeError) as exc:
        print(f"cannot read observation stream {path}: {exc}", file=sys.stderr)
        return 1

    if target_actions.count("run") != 1:
        print(
            f"exact target must have one run event; observed {target_actions!r}",
            file=sys.stderr,
        )
        return 1
    if terminal_actions != ["pass"]:
        print(
            "exact target must have one passing terminal event; "
            f"observed {terminal_actions!r}",
            file=sys.stderr,
        )
        return 1
    if target_actions.index("run") > target_actions.index("pass"):
        print(
            f"exact target pass preceded run event: {target_actions!r}",
            file=sys.stderr,
        )
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
