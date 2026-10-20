#!/usr/bin/env python3
"""Check retained license bytes and coverage of required Go modules, offline."""

import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys


def check(root):
    errors = []
    license_dir = root / "licenses"
    rows = re.findall(
        r"^\| \[[^\]]+\]\(([^)]+)\) \|.*\| `([0-9a-f]{64})` \|$",
        (license_dir / "README.md").read_text(),
        re.MULTILINE,
    )
    if not rows:
        errors.append("licenses/README.md contains no original-license hashes")
    retained = set()
    for relative, expected in rows:
        path = (license_dir / relative).resolve()
        if not path.is_relative_to(root):
            errors.append(f"license path escapes repository: {relative}")
            continue
        if path in retained:
            errors.append(f"duplicate retained license entry: {relative}")
        retained.add(path)
        if not path.is_file():
            errors.append(f"missing retained license: {relative}")
        elif hashlib.sha256(path.read_bytes()).hexdigest() != expected:
            errors.append(f"retained license hash changed: {relative}")

    for path in license_dir.iterdir():
        if path.is_file() and ("LICENSE" in path.name or "NOTICE" in path.name):
            if path.resolve() not in retained:
                errors.append(f"retained license is absent from hash inventory: {path.name}")

    required = json.loads(subprocess.check_output(
        ["go", "mod", "edit", "-json"], cwd=root, text=True, timeout=30
    ))
    coverage = json.loads((license_dir / "dependencies.json").read_text())
    dependencies = {item["Path"]: item["Version"] for item in required.get("Require", [])}
    for module, version in dependencies.items():
        entry = coverage.get(module)
        if entry is None:
            errors.append(f"unreviewed dependency: {module}@{version}")
            continue
        if entry.get("version") != version:
            errors.append(f"dependency license review version changed: {module}@{version}")
        files = entry.get("licenses", [])
        if not files:
            errors.append(f"dependency has no retained licenses: {module}")
        for filename in files:
            if (license_dir / filename).resolve() not in retained:
                errors.append(f"dependency license is absent from hash inventory: {filename}")
    for module in coverage.keys() - dependencies.keys():
        errors.append(f"stale dependency license mapping: {module}")
    if required.get("Replace"):
        errors.append("go.mod replacements require a separate dependency license review")

    fork_license = root / "internal/jmespath/LICENSE"
    upstream_fork_license = license_dir / "go-jmespath-v0.4.0-LICENSE"
    if not fork_license.is_file() or fork_license.read_bytes() != upstream_fork_license.read_bytes():
        errors.append("internal/jmespath/LICENSE differs from retained upstream notice")
    for filename in ("NOTICE", "THIRD_PARTY_NOTICES.md", "docs/licensing.md"):
        if not (root / filename).is_file() or not (root / filename).read_text().strip():
            errors.append(f"missing distribution notice: {filename}")
    if errors:
        for error in errors:
            print(error, file=sys.stderr)
        return 1
    print(f"Licensing check passed: {len(retained)} unchanged originals, "
          f"{len(dependencies)} pinned Go modules, fork and distribution notices.")
    return 0


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[3])
    args = parser.parse_args()
    try:
        return check(args.root.resolve())
    except (OSError, ValueError, subprocess.SubprocessError) as error:
        print(f"Licensing check failed: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
