from __future__ import annotations

import importlib.util
import tempfile
import unittest
from pathlib import Path

MODULE_PATH = Path(__file__).with_name("validate_manifest_paths.py")
SPEC = importlib.util.spec_from_file_location("validate_manifest_paths", MODULE_PATH)
assert SPEC is not None and SPEC.loader is not None
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class ManifestPathValidationTests(unittest.TestCase):
    def test_accepts_regular_manifest_inside_repository(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            router = root / ".repora" / "document-router.yaml"
            manifest = root / "docs" / "routing" / "router.manifest.yaml"
            router.parent.mkdir()
            manifest.parent.mkdir(parents=True)
            router.write_text(
                "version: 1\nmanifests:\n  - docs/routing/router.manifest.yaml\ndefaults:\n",
                encoding="utf-8",
            )
            manifest.write_text("version: 1\n", encoding="utf-8")

            self.assertEqual(MODULE.validate_manifest_paths(router), [manifest.resolve()])

    def test_rejects_symlink_escape(self) -> None:
        with tempfile.TemporaryDirectory() as tmp, tempfile.TemporaryDirectory() as outside:
            root = Path(tmp)
            router = root / ".repora" / "document-router.yaml"
            link = root / "docs" / "routing" / "router.manifest.yaml"
            external = Path(outside) / "external.yaml"
            router.parent.mkdir()
            link.parent.mkdir(parents=True)
            external.write_text("version: 1\n", encoding="utf-8")
            link.symlink_to(external)
            router.write_text(
                "version: 1\nmanifests:\n  - docs/routing/router.manifest.yaml\ndefaults:\n",
                encoding="utf-8",
            )

            with self.assertRaisesRegex(ValueError, "resolves outside repository"):
                MODULE.validate_manifest_paths(router)

    def test_rejects_parent_traversal(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            router = root / ".repora" / "document-router.yaml"
            router.parent.mkdir()
            router.write_text(
                "version: 1\nmanifests:\n  - ../outside.yaml\ndefaults:\n",
                encoding="utf-8",
            )

            with self.assertRaisesRegex(ValueError, "unsafe manifest path"):
                MODULE.validate_manifest_paths(router)


if __name__ == "__main__":
    unittest.main()
