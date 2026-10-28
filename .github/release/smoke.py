#!/usr/bin/env python3
"""Build release archives and test npm installation without an agent login."""

import hashlib
import json
import os
from pathlib import Path
import shutil
import sys
import tarfile
import tempfile
import zipfile

REPO = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(REPO / ".claude/skills/verify-paws/scripts"))
from drive import Installer, sh


def run(command, project, env):
    result = sh(command, project, env, timeout=300)
    if result.returncode:
        raise RuntimeError(f"{command[0]} failed:\n{result.stdout}{result.stderr}")
    return result.stdout.strip()


def main():
    version = json.loads((REPO / "npm/package.json").read_text())["version"]
    with tempfile.TemporaryDirectory(prefix="paws-package-smoke-") as temporary:
        scratch = Path(temporary)
        installer = Installer.npm(scratch)
        try:
            dist = scratch / "dist"
            sums_path = dist / "checksums.txt"
            original_sums = sums_path.read_text()
            sums = {name.strip(): digest for digest, name in
                    (line.split(maxsplit=1) for line in original_sums.splitlines() if line.strip())}
            for goos in ["darwin", "linux", "windows"]:
                for arch in ["amd64", "arm64"]:
                    extension = "zip" if goos == "windows" else "tar.gz"
                    asset = f"paws_{version}_{goos}_{arch}.{extension}"
                    archive = dist / asset
                    if hashlib.sha256(archive.read_bytes()).hexdigest() != sums[asset]:
                        raise RuntimeError(f"Checksum mismatch for {asset}")
                    if goos == "windows":
                        with zipfile.ZipFile(archive) as bundle:
                            members = bundle.namelist()
                        binary = "paws.exe"
                    else:
                        with tarfile.open(archive) as bundle:
                            members = bundle.getnames()
                        binary = "paws"
                    if not {binary, "LICENSE", "README.md"}.issubset(members):
                        raise RuntimeError(f"Missing archive contents in {asset}")
            print("PASS: all six archives, contents, and SHA-256 checksums", flush=True)

            project = scratch / "project"
            project.mkdir()
            env = {**os.environ, **installer.env,
                   "PAWS_HOME": str(scratch / "paws-home"),
                   "npm_config_cache": str(scratch / "npm-cache"),
                   "npm_config_audit": "false", "npm_config_fund": "false"}
            run(["git", "init", "-q"], project, env)
            version_command = [*installer.init[:-1], "version"]
            if run(version_command, project, env) != f"paws {version}":
                raise RuntimeError("npm wrapper ran the wrong version")
            run(installer.init, project, env)
            settings = json.loads((project / ".claude/settings.json").read_text())
            command = settings["hooks"]["PreToolUse"][0]["hooks"][0]["command"]
            if "PAWS_MANAGED=1" not in command or "paws-home" not in command:
                raise RuntimeError("Hook does not point to the installed binary")
            binary = scratch / "paws-home/bin" / ("paws.exe" if sys.platform == "win32" else "paws")
            shutil.rmtree(scratch / "npm-cache")
            if run([str(binary), "version"], project, env) != f"paws {version}":
                raise RuntimeError("Installed binary failed after clearing the npm cache")
            print("PASS: fresh npm install, init, hook registration, and cache-independent binary", flush=True)

            sums_path.write_text("\n".join(f"{'0' * 64}  {name}" for name in sums) + "\n")
            rejected = sh(version_command, project, env, timeout=300)
            if rejected.returncode == 0 or "checksum mismatch" not in rejected.stderr:
                raise RuntimeError("npm wrapper accepted corrupted checksums")
            sums_path.write_text(original_sums)
            if run(version_command, project, env) != f"paws {version}":
                raise RuntimeError("npm wrapper did not recover after restoring checksums")
            print("PASS: corrupted checksums rejected, retry succeeds after repair", flush=True)
        finally:
            installer.cleanup()


if __name__ == "__main__":
    main()
