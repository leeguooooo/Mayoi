#!/usr/bin/env python3
"""Exercise release confirmation using only temporary local Git repositories."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


@unittest.skipUnless(os.name == "posix", "release script needs a POSIX shell")
class ReleaseConfirmation(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="mayoi-release-test-")
        self.addCleanup(self.temp.cleanup)
        root = Path(self.temp.name)
        self.repo = root / "repo"
        self.remote = root / "remote.git"
        self.repo.mkdir()
        scripts = self.repo / "scripts"
        scripts.mkdir()
        source = Path(__file__).resolve().parent
        for name in ("bump.sh", "release-notes.sh"):
            shutil.copy2(source / name, scripts / name)
        (self.repo / "mygo.json").write_text('{\n  "version": "1.2.3"\n}\n')
        (self.repo / "CHANGELOG.md").write_text("# Changelog\n\n## Unreleased\n\n- A change.\n")
        tools = root / "bin"
        tools.mkdir()
        gh = tools / "gh"
        gh.write_text("#!/bin/sh\n# No release exists; never contact GitHub.\nexit 1\n")
        gh.chmod(0o755)
        self.env = dict(os.environ, PATH=str(tools) + os.pathsep + os.environ["PATH"],
                        GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_SYSTEM=os.devnull)
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.name", "Release Test")
        self.git("config", "user.email", "release-test@example.invalid")
        self.git("config", "commit.gpgsign", "false")
        self.git("init", "-q", "--bare", str(self.remote))
        self.git("add", ".")
        self.git("commit", "-q", "-m", "Fixture")
        self.git("remote", "add", "origin", str(self.remote))
        self.git("push", "-q", "origin", "main")
        self.head = self.git("rev-parse", "HEAD")

    def git(self, *args):
        return subprocess.check_output(["git", *args], cwd=self.repo, env=self.env,
                                       text=True, stderr=subprocess.PIPE).strip()

    def run_bump(self, *args, answer=None):
        if answer is None:
            return subprocess.run(["bash", "scripts/bump.sh", *args], cwd=self.repo,
                                  env=self.env, stdin=subprocess.DEVNULL,
                                  capture_output=True, text=True, timeout=15)
        import pty
        master, slave = pty.openpty()
        try:
            proc = subprocess.Popen(["bash", "scripts/bump.sh", *args], cwd=self.repo,
                                    env=self.env, stdin=slave, stdout=subprocess.PIPE,
                                    stderr=subprocess.PIPE, text=True)
            os.write(master, (answer + "\n").encode())
            try:
                stdout, stderr = proc.communicate(timeout=15)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.communicate()
                raise
            return subprocess.CompletedProcess(proc.args, proc.returncode, stdout, stderr)
        finally:
            os.close(master)
            os.close(slave)

    def assert_unchanged(self):
        self.assertEqual(self.git("rev-parse", "HEAD"), self.head)
        self.assertEqual(self.git("status", "--porcelain"), "")
        self.assertEqual(self.git("tag"), "")
        self.assertEqual(self.git("ls-remote", "--tags", "origin"), "")
        self.assertEqual(self.git("ls-remote", "origin", "refs/heads/main").split()[0], self.head)

    def assert_published(self):
        version = json.loads((self.repo / "mygo.json").read_text())["version"]
        self.assertEqual(version, "1.2.4")
        self.assertIn("## 1.2.4\n", (self.repo / "CHANGELOG.md").read_text())
        head = self.git("rev-parse", "HEAD")
        self.assertNotEqual(head, self.head)
        self.assertEqual(self.git("rev-parse", "v1.2.4"), head)
        self.assertEqual(self.git("ls-remote", "origin", "refs/tags/v1.2.4").split()[0], head)
        self.assertEqual(self.git("ls-remote", "origin", "refs/heads/main").split()[0], head)
        self.assertEqual(self.git("status", "--porcelain"), "")

    def test_noninteractive_requires_yes(self):
        result = self.run_bump("patch")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("--yes", result.stderr)
        self.assert_unchanged()

    def test_decline_leaves_everything_unchanged(self):
        result = self.run_bump("patch", answer="n")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("Cancelled", result.stdout)
        self.assert_unchanged()

    def test_confirm_publishes(self):
        result = self.run_bump("patch", answer="yes")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assert_published()

    def test_explicit_yes_publishes_without_terminal(self):
        result = self.run_bump("--yes", "patch")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assert_published()


if __name__ == "__main__":
    unittest.main()
