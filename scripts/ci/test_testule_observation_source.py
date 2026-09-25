#!/usr/bin/env python3
import json
import os
import stat
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent
CAPTURE = ROOT / "capture-json-stream.py"
VERIFY = ROOT / "verify-go-test-observation.py"


class CaptureJSONStreamTests(unittest.TestCase):
    def run_capture(self, data, destination, limit):
        return subprocess.run(
            [sys.executable, str(CAPTURE), str(destination), str(limit)],
            input=data,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )

    def test_exact_bound_is_retained_privately(self):
        with tempfile.TemporaryDirectory() as tmp:
            destination = Path(tmp) / "stream.json"
            result = self.run_capture(b"abcdef", destination, 6)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(destination.read_bytes(), b"abcdef")
            self.assertEqual(stat.S_IMODE(destination.stat().st_mode) & 0o077, 0)

    def test_overflow_drains_but_marks_retained_prefix_untrusted(self):
        with tempfile.TemporaryDirectory() as tmp:
            destination = Path(tmp) / "stream.json"
            result = self.run_capture(b"abcdefgh", destination, 5)
            self.assertEqual(result.returncode, 3)
            self.assertEqual(destination.read_bytes(), b"abcde")
            self.assertIn(b"must not be used as evidence input", result.stderr)

    def test_existing_destination_is_not_overwritten(self):
        with tempfile.TemporaryDirectory() as tmp:
            destination = Path(tmp) / "stream.json"
            destination.write_bytes(b"existing")
            result = self.run_capture(b"replacement", destination, 32)
            self.assertEqual(result.returncode, 1)
            self.assertEqual(destination.read_bytes(), b"existing")


class VerifyGoTestObservationTests(unittest.TestCase):
    package = "repoctl/internal/plan"
    target = "TestReconcileIsDeterministicAndDoesNotMutateInputs"

    def run_verify(self, events):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "stream.json"
            path.write_text(
                "".join(json.dumps(event) + "\n" for event in events),
                encoding="utf-8",
            )
            return subprocess.run(
                [sys.executable, str(VERIFY), str(path), self.package, self.target],
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                check=False,
            )

    def event(self, action, package=None, target=None):
        item = {
            "Action": action,
            "Package": self.package if package is None else package,
        }
        if target is not False:
            item["Test"] = self.target if target is None else target
        return item

    def test_exact_run_and_pass_are_accepted(self):
        result = self.run_verify([
            self.event("run"),
            self.event("output"),
            self.event("pass"),
            self.event("pass", target=False),
        ])
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_skip_fail_missing_and_mismatch_are_rejected(self):
        cases = {
            "skip": [self.event("run"), self.event("skip")],
            "fail": [self.event("run"), self.event("fail")],
            "missing": [self.event("pass", target=False)],
            "mismatch": [
                self.event("run", target="OtherTest"),
                self.event("pass", target="OtherTest"),
            ],
        }
        for name, events in cases.items():
            with self.subTest(name=name):
                result = self.run_verify(events)
                self.assertEqual(result.returncode, 1)

    def test_malformed_json_is_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "stream.json"
            path.write_text("{not-json}\n", encoding="utf-8")
            result = subprocess.run(
                [sys.executable, str(VERIFY), str(path), self.package, self.target],
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                check=False,
            )
            self.assertEqual(result.returncode, 1)
            self.assertIn(b"invalid JSON", result.stderr)


if __name__ == "__main__":
    unittest.main()
