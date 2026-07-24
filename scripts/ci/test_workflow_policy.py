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


if __name__ == '__main__':
    unittest.main()
