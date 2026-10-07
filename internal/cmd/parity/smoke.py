#!/usr/bin/env python3
"""Run bounded core-user HTTP workflows and save their actual test results."""

import argparse
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys


FLOWS = [
    ("server-network", "서버·네트워크", [
        (".", "TestConnectionAutomaticCreateOwnsNamedDependencyPagesAndLifetime"),
        (".", "TestConnectionServerFloatingIPOrdersWaitsAndPreservesPartialResults"),
        (".", "TestConnectionNetworkMutationRefreshesGetterDefaultNICAndFloatingRoles"),
    ]),
    ("volume", "볼륨 생성·연결·기본 대기", [
        ("./blockstorage", "TestCreateVolumeContractsDefaultFreshWaitAndIndependentOwnedResults"),
        ("./blockstorage", "TestAttachVolumeContractsDefaultSequenceAndIndependentProof"),
    ]),
    ("image", "이미지 import 접수·checksum 다운로드", [
        ("./image", "TestCreateAndImportDefaultDirectSequenceAndEvidence"),
        ("./image", "TestDownloadToDefaultSequenceAndEvidence"),
    ]),
    ("secret", "Secret metadata·payload 조회", [
        (".", "TestConnectionKeyManagerSecretFetchSharedClientAndLiveToken"),
    ]),
    ("object", "Swift object 요청·checksum 기반 업로드 생략", [
        (".", "TestConnectionObjectCreateSharesClientAndOwnsSources"),
    ]),
]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--report", type=Path, default=Path(".reports/core-smoke.json"))
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[3]
    module = re.search(r"^module\s+(\S+)", (root / "go.mod").read_text(), re.M).group(1)
    cases = [case for _, _, tests in FLOWS for case in tests]
    packages = list(dict.fromkeys(package for package, _ in cases))
    pattern = "^(" + "|".join(re.escape(test) for _, test in cases) + ")$"
    command = ["go", "test", "-race", "-count=1", "-timeout", "60s", "-json", *packages, "-run", pattern]
    digest = hashlib.sha256()
    for path in sorted(root.rglob("*.go")):
        digest.update(str(path.relative_to(root)).encode() + b"\0" + path.read_bytes() + b"\0")
    report_path = args.report if args.report.is_absolute() else root / args.report
    report_path.parent.mkdir(parents=True, exist_ok=True)
    log_path = report_path.with_suffix(".log")
    events = {}
    with log_path.open("w") as log:
        process = subprocess.Popen(command, cwd=root, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        for line in process.stdout:
            log.write(line)
            try:
                event = json.loads(line)
            except ValueError:
                continue
            if event.get("Test") and event.get("Action") in {"pass", "fail", "skip"}:
                events[(event["Package"], event["Test"])] = event["Action"]
        code = process.wait()
    flows = []
    for identity, label, tests in FLOWS:
        results = []
        for package, test in tests:
            package_name = module if package == "." else module + package[1:]
            results.append({"package": package_name, "test": test, "status": events.get((package_name, test), "missing")})
        passed = code == 0 and all(result["status"] == "pass" for result in results)
        flows.append({"id": identity, "label": label, "status": "PASS" if passed else "FAIL", "tests": results})
        print(f"{flows[-1]['status']}: {label} ({len(tests)} groups)", flush=True)
    passed = code == 0 and all(flow["status"] == "PASS" for flow in flows)
    report = {
        "scope": "local HTTP fixtures; no authenticated OpenStack execution",
        "tested_at": datetime.now(timezone.utc).isoformat(),
        "go_source_sha256": digest.hexdigest(), "command": command,
        "exit_code": code, "status": "PASS" if passed else "FAIL", "flows": flows,
    }
    report_path.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(f"Report: {report_path}", flush=True)
    if not passed:
        print(f"Failure details: {log_path}", file=sys.stderr)
    return 0 if passed else 1


if __name__ == "__main__":
    sys.exit(main())
