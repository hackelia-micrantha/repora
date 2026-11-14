#!/usr/bin/env python3
from pathlib import Path
import argparse
import re
import sys

DEFAULT_WORKFLOW_DIR = Path('.github/workflows')
SHA_REF = re.compile(r'^\s*-?\s*uses:\s*([^\s]+)@([0-9a-f]{40})\s+#\s+\S.*$')
USES = re.compile(r'^\s*-?\s*uses:\s*([^\s]+)@([^\s#]+)')
JOB = re.compile(r'^  ([A-Za-z0-9_-]+):\s*$')
TIMEOUT = re.compile(r'^    timeout-minutes:\s*\d+\s*$')
REUSABLE_JOB = re.compile(r'^    uses:\s*[^\s]+@[^\s#]+(?:\s+#.*)?

def validate(workflow_dir: Path) -> list[str]:
    errors: list[str] = []
    files = sorted(workflow_dir.glob('*.y*ml'))
    if not files:
        return ['no workflow files found']

    for path in files:
        lines = path.read_text(encoding='utf-8').splitlines()
        if contains_pull_request_target(lines):
            errors.append(f'{path}: pull_request_target is prohibited')
        if not any(line == 'permissions:' for line in lines):
            errors.append(f'{path}: top-level permissions block is required')

        in_jobs = False
        current_job: str | None = None
        job_has_timeout = False
        job_delegates_reusable_workflow = False

        def finish_job() -> None:
            if current_job and not job_has_timeout and not job_delegates_reusable_workflow:
                errors.append(f'{path}: job {current_job!r} lacks timeout-minutes')

        for number, line in enumerate(lines, start=1):
            if line == 'jobs:':
                in_jobs = True
                continue
            if in_jobs:
                match = JOB.match(line)
                if match:
                    finish_job()
                    current_job = match.group(1)
                    job_has_timeout = False
                    job_delegates_reusable_workflow = False
                elif current_job and TIMEOUT.match(line):
                    job_has_timeout = True
                elif current_job and REUSABLE_JOB.match(line):
                    # GitHub does not allow timeout-minutes on a caller job whose
                    # top-level uses delegates to a reusable workflow. The called
                    # workflow must own the actual runner-job timeout instead.
                    job_delegates_reusable_workflow = True

            uses = USES.match(line)
            if not uses:
                continue
            action, ref = uses.groups()
            if action.startswith('./'):
                continue
            if not SHA_REF.match(line):
                errors.append(
                    f'{path}:{number}: third-party action {action}@{ref} must use a full SHA and version comment'
                )

        finish_job()

    return errors


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument('--workflow-dir', type=Path, default=DEFAULT_WORKFLOW_DIR)
    args = parser.parse_args()

    errors = validate(args.workflow_dir)
    if errors:
        for error in errors:
            print(error, file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
)

ON_KEY = re.compile(r"^(?:on|'on'|\"on\")\s*:\s*(.*)$")
FORBIDDEN_EVENT = re.compile(r"(?<![A-Za-z0-9_-])pull_request_target(?![A-Za-z0-9_-])")


def contains_pull_request_target(lines: list[str]) -> bool:
    """Inspect the top-level 'on' trigger, not incidental text in job scripts."""
    for index, line in enumerate(lines):
        match = ON_KEY.match(line)
        if match is None:
            continue
        value = match.group(1).split(' #', 1)[0].strip()
        if value:
            if FORBIDDEN_EVENT.search(value):
                return True
            # A flow-style sequence or mapping can continue on later lines.
            if value.startswith(('[', '{')) and not any(ch in value for ch in ']}'):
                for entry in lines[index + 1:]:
                    if FORBIDDEN_EVENT.search(entry.split(' #', 1)[0]):
                        return True
                    if any(ch in entry for ch in ']}'):
                        break
            return False

        event_indent = None
        for entry in lines[index + 1:]:
            if not entry.strip() or entry.lstrip().startswith('#'):
                continue
            indent = len(entry) - len(entry.lstrip(' '))
            if indent == 0:
                break
            if event_indent is None:
                event_indent = indent
            if indent != event_indent:
                continue
            token = entry.strip().removeprefix('-').split(':', 1)[0].strip("'\" ,]")
            if token == 'pull_request_target':
                return True
        return False
    return False


def validate(workflow_dir: Path) -> list[str]:
    errors: list[str] = []
    files = sorted(workflow_dir.glob('*.y*ml'))
    if not files:
        return ['no workflow files found']

    for path in files:
        lines = path.read_text(encoding='utf-8').splitlines()
        text = '\n'.join(lines)
        if 'pull_request_target:' in text:
            errors.append(f'{path}: pull_request_target is prohibited')
        if not any(line == 'permissions:' for line in lines):
            errors.append(f'{path}: top-level permissions block is required')

        in_jobs = False
        current_job: str | None = None
        job_has_timeout = False
        job_delegates_reusable_workflow = False

        def finish_job() -> None:
            if current_job and not job_has_timeout and not job_delegates_reusable_workflow:
                errors.append(f'{path}: job {current_job!r} lacks timeout-minutes')

        for number, line in enumerate(lines, start=1):
            if line == 'jobs:':
                in_jobs = True
                continue
            if in_jobs:
                match = JOB.match(line)
                if match:
                    finish_job()
                    current_job = match.group(1)
                    job_has_timeout = False
                    job_delegates_reusable_workflow = False
                elif current_job and TIMEOUT.match(line):
                    job_has_timeout = True
                elif current_job and REUSABLE_JOB.match(line):
                    # GitHub does not allow timeout-minutes on a caller job whose
                    # top-level uses delegates to a reusable workflow. The called
                    # workflow must own the actual runner-job timeout instead.
                    job_delegates_reusable_workflow = True

            uses = USES.match(line)
            if not uses:
                continue
            action, ref = uses.groups()
            if action.startswith('./'):
                continue
            if not SHA_REF.match(line):
                errors.append(
                    f'{path}:{number}: third-party action {action}@{ref} must use a full SHA and version comment'
                )

        finish_job()

    return errors


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument('--workflow-dir', type=Path, default=DEFAULT_WORKFLOW_DIR)
    args = parser.parse_args()

    errors = validate(args.workflow_dir)
    if errors:
        for error in errors:
            print(error, file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
