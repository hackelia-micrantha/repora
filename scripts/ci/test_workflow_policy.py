#!/usr/bin/env python3
from pathlib import Path
import subprocess
import tempfile
import textwrap
import unittest

SCRIPT = Path(__file__).with_name('workflow-policy.py')
SHA = '1' * 40


class WorkflowPolicyTest(unittest.TestCase):
    def run_policy(self, workflow: str) -> subprocess.CompletedProcess[str]:
        with tempfile.TemporaryDirectory() as temp_dir:
            workflow_dir = Path(temp_dir)
            (workflow_dir / 'test.yml').write_text(textwrap.dedent(workflow), encoding='utf-8')
            return subprocess.run(
                ['python3', str(SCRIPT), '--workflow-dir', str(workflow_dir)],
                check=False,
                capture_output=True,
                text=True,
            )

    def test_accepts_pinned_action_with_version_comment(self) -> None:
        result = self.run_policy(
            f'''\
            name: test
            on: push
            permissions:
              contents: read
            jobs:
              verify:
                runs-on: ubuntu-latest
                timeout-minutes: 5
                steps:
                  - uses: actions/checkout@{SHA} # v4
            '''
        )
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_accepts_pinned_reusable_workflow_without_caller_timeout(self) -> None:
        result = self.run_policy(
            f'''\
            name: test
            on: push
            permissions:
              contents: read
            jobs:
              delegated:
                uses: example/shared/.github/workflows/ci.yml@{SHA} # automation-v1 candidate
                with:
                  command: test
            '''
        )
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_rejects_pinned_action_without_version_comment(self) -> None:
        result = self.run_policy(
            f'''\
            name: test
            on: push
            permissions:
              contents: read
            jobs:
              verify:
                runs-on: ubuntu-latest
                timeout-minutes: 5
                steps:
                  - uses: actions/checkout@{SHA}
            '''
        )
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('must use a full SHA and version comment', result.stderr)

    def test_rejects_reusable_workflow_without_version_comment(self) -> None:
        result = self.run_policy(
            f'''\
            name: test
            on: push
            permissions:
              contents: read
            jobs:
              delegated:
                uses: example/shared/.github/workflows/ci.yml@{SHA}
            '''
        )
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('must use a full SHA and version comment', result.stderr)

    def test_rejects_mutable_action_ref(self) -> None:
        result = self.run_policy(
            '''\
            name: test
            on: push
            permissions:
              contents: read
            jobs:
              verify:
                runs-on: ubuntu-latest
                timeout-minutes: 5
                steps:
                  - uses: actions/checkout@v4 # v4
            '''
        )
        self.assertNotEqual(result.returncode, 0)

    def test_rejects_missing_timeout(self) -> None:
        result = self.run_policy(
            f'''\
            name: test
            on: push
            permissions:
              contents: read
            jobs:
              verify:
                runs-on: ubuntu-latest
                steps:
                  - uses: actions/checkout@{SHA} # v4
            '''
        )
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('lacks timeout-minutes', result.stderr)


    def test_rejects_pull_request_target_trigger_forms(self) -> None:
        triggers = (
            'on: pull_request_target',
            'on: [push, pull_request_target]',
            'on: [\n  push,\n  pull_request_target\n]',
            'on: {pull_request_target: {types: [opened]}}',
            'on: {\n  push: {types: [opened]},\n  pull_request_target: {types: [opened]}\n}',
            'on:\n  pull_request:\n  pull_request_target:',
            'on:\n  - push\n  - pull_request_target',
            '"on": "pull_request_target"',
        )
        for trigger in triggers:
            with self.subTest(trigger=trigger):
                result = self.run_policy(
                    'name: test\n' + trigger + '\n'
                    'permissions:\n  contents: read\n'
                    'jobs:\n  verify:\n    runs-on: ubuntu-latest\n'
                    '    timeout-minutes: 5\n    steps:\n'
                    '      - run: echo ok\n'
                )
                self.assertNotEqual(result.returncode, 0, result.stderr)
                self.assertIn('pull_request_target is prohibited', result.stderr)

    def test_does_not_treat_comments_or_job_text_as_triggers(self) -> None:
        for trigger in ('on: push # pull_request_target is prohibited',
                        'on: workflow_dispatch'):
            with self.subTest(trigger=trigger):
                result = self.run_policy(
                    'name: test\n' + trigger + '\n'
                    'permissions:\n  contents: read\n'
                    'jobs:\n  verify:\n    runs-on: ubuntu-latest\n'
                    '    timeout-minutes: 5\n    steps:\n'
                    '      - run: echo pull_request_target\n'
                )
                self.assertEqual(result.returncode, 0, result.stderr)

    def test_nested_workflow_call_input_description_does_not_trigger(self) -> None:
        result = self.run_policy(
            'name: test\non:\n  workflow_call:\n    inputs:\n'
            '      label:\n        description: pull_request_target\n'
            '        required: false\n        type: string\n'
            'permissions:\n  contents: read\n'
            'jobs:\n  verify:\n    runs-on: ubuntu-latest\n'
            '    timeout-minutes: 5\n    steps:\n'
            '      - run: echo ok\n'
        )
        self.assertEqual(result.returncode, 0, result.stderr)



if __name__ == '__main__':
    unittest.main()
