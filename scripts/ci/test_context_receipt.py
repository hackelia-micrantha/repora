#!/usr/bin/env python3
"""Regression tests for the context receipt validation contract."""

from __future__ import annotations

import copy
import importlib.util
import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
VALIDATOR_PATH = Path(__file__).with_name("context-receipt.py")
EXAMPLE_PATH = ROOT / "examples" / "context-receipt-v1.json"

SPEC = importlib.util.spec_from_file_location("context_receipt", VALIDATOR_PATH)
if SPEC is None or SPEC.loader is None:
    raise RuntimeError("could not load context receipt validator")
context_receipt = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(context_receipt)


class ContextReceiptTests(unittest.TestCase):
    def setUp(self) -> None:
        self.receipt = json.loads(EXAMPLE_PATH.read_text(encoding="utf-8"))

    def assert_invalid(self, receipt: dict, message: str) -> None:
        with self.assertRaisesRegex(ValueError, message):
            context_receipt.validate(receipt)

    def test_example_is_canonical_and_valid(self) -> None:
        raw = EXAMPLE_PATH.read_text(encoding="utf-8")
        digest = context_receipt.validate(self.receipt, raw)
        self.assertRegex(digest, r"^[0-9a-f]{64}$")

    def test_noncanonical_serialization_is_rejected(self) -> None:
        raw = json.dumps(self.receipt, indent=4) + "\n"
        with self.assertRaisesRegex(ValueError, "canonical deterministic"):
            context_receipt.validate(self.receipt, raw)

    def test_credential_like_query_is_rejected(self) -> None:
        receipt = copy.deepcopy(self.receipt)
        header = "author" + "ization"
        scheme = "bear" + "er"
        receipt["query"] = f"{header}: {scheme} example-value"
        self.assert_invalid(receipt, "credential-like")

    def test_credential_bearing_url_is_rejected(self) -> None:
        receipt = copy.deepcopy(self.receipt)
        authority = "example-user" + ":" + "example-value" + "@"
        receipt["selected"][0]["reason"] = (
            "from " + "https://" + authority + "example.test/input"
        )
        self.assert_invalid(receipt, "credential-bearing URL")

    def test_unsafe_backslash_path_is_rejected(self) -> None:
        receipt = copy.deepcopy(self.receipt)
        receipt["selected"][0]["path"] = "docs\\private.md"
        self.assert_invalid(receipt, "safe repository-relative POSIX path")

    def test_oversized_snippet_is_rejected(self) -> None:
        receipt = copy.deepcopy(self.receipt)
        receipt["selected"][0]["snippets"][0]["bytes"] = 4097
        self.assert_invalid(receipt, "outside the v1 bound")

    def test_budget_mismatch_is_rejected(self) -> None:
        receipt = copy.deepcopy(self.receipt)
        receipt["budget"]["selected_bytes"] += 1
        self.assert_invalid(receipt, "does not match selected input bytes")

    def test_fallback_receipt_is_supported(self) -> None:
        receipt = copy.deepcopy(self.receipt)
        receipt["routing"]["routes"] = []
        receipt["routing"]["fallback"] = "docs-first"
        digest = context_receipt.validate(receipt)
        self.assertRegex(digest, r"^[0-9a-f]{64}$")

    def test_routes_and_fallback_cannot_both_be_selected(self) -> None:
        receipt = copy.deepcopy(self.receipt)
        receipt["routing"]["fallback"] = "docs-first"
        self.assert_invalid(receipt, "either selected routes or one fallback")

    def test_duplicate_snippet_indices_are_rejected(self) -> None:
        receipt = copy.deepcopy(self.receipt)
        duplicate = copy.deepcopy(receipt["selected"][0]["snippets"][0])
        receipt["selected"][0]["snippets"].append(duplicate)
        self.assert_invalid(receipt, "snippet indices must be unique")


if __name__ == "__main__":
    unittest.main()
