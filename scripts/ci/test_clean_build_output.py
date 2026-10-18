"""Bounded native build cleanup regression tests."""

from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


SCRIPT = Path(__file__).with_name("clean-build-output.sh")
BINARIES = (
    "repoctl",
    "repoctl-linux-amd64",
    "repoctl-windows-amd64.exe",
    "repoctl-darwin-amd64",
    "repoctl-darwin-arm64",
)


class CleanBuildOutputTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="repora-build-clean-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        scripts = self.root / "scripts" / "ci"
        scripts.mkdir(parents=True)
        self.command = scripts / "clean-build-output.sh"
        shutil.copyfile(SCRIPT, self.command)

    def run_clean(self):
        return subprocess.run(
            ["bash", str(self.command)],
            cwd=self.root, capture_output=True, text=True, check=False,
        )

    def test_missing_build_outputs_is_idempotent(self):
        self.assertEqual(self.run_clean().returncode, 0)
        self.assertEqual(self.run_clean().returncode, 0)

    def test_removes_only_known_generated_binary_names(self):
        binaries = self.root / "bin"
        binaries.mkdir()
        files = [self.root / "repoctl"] + [binaries / name for name in BINARIES]
        for path in files:
            path.write_text("test generated executable")
        keep = [
            binaries / "custom-tool",
            self.root / "artifacts" / "coverage" / "coverage.out",
            self.root / "artifacts" / "testule" / "observation.json",
            self.root / "dist" / "release.tar.gz",
        ]
        for path in keep:
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text("must retain")
        self.assertEqual(self.run_clean().returncode, 0)
        for path in files:
            self.assertFalse(path.exists(), path)
        for path in keep:
            self.assertEqual(path.read_text(), "must retain")
        self.assertEqual(self.run_clean().returncode, 0)

    def test_rejects_symlinked_bin_directory(self):
        external = self.root / "external"
        external.mkdir()
        (external / "repoctl").write_text("external")
        (self.root / "bin").symlink_to(external, target_is_directory=True)
        result = self.run_clean()
        self.assertEqual(result.returncode, 2)
        self.assertIn("symlinked bin directory", result.stderr)
        self.assertEqual((external / "repoctl").read_text(), "external")

    def test_rejects_symlinked_binary_without_partial_cleanup(self):
        binaries = self.root / "bin"
        binaries.mkdir()
        external = self.root / "external"
        external.write_text("outside")
        (self.root / "repoctl").write_text("preserve on validation failure")
        (binaries / "repoctl").symlink_to(external)
        result = self.run_clean()
        self.assertEqual(result.returncode, 2)
        self.assertIn("symlinked build output", result.stderr)
        self.assertTrue((self.root / "repoctl").exists())
        self.assertEqual(external.read_text(), "outside")

    def test_rejects_non_file_output(self):
        binaries = self.root / "bin"
        binaries.mkdir()
        (binaries / "repoctl").mkdir()
        result = self.run_clean()
        self.assertEqual(result.returncode, 2)
        self.assertIn("non-file build output", result.stderr)


if __name__ == "__main__":
    unittest.main()
