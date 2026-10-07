#!/usr/bin/env python3
"""Refresh declared API progress from the durable catalog and reviews."""

import argparse
from collections import Counter
import json
from pathlib import Path
import sys


START = "<!-- sdk-progress:start -->"
END = "<!-- sdk-progress:end -->"
STATUSES = {"supported", "go_mapping", "unsupported", "unresolved"}


def summary(root):
    catalog = json.loads((root / "api/sdk_support_catalog.json").read_text())
    ledger = json.loads((root / "api/sdk_reviews.json").read_text())
    if catalog["schema_version"] != 1 or ledger["schema_version"] != 1:
        raise ValueError("unsupported progress schema")
    if catalog["source_pins"] != ledger["source_pins"]:
        raise ValueError("catalog and review source pins differ")
    operations = catalog["operations"]
    counts = Counter(unresolved=len(operations))
    seen = set()
    contracts = 0
    for review in ledger["reviews"]:
        operation = review["operation"]
        if operation in seen or operation not in operations:
            raise ValueError(f"duplicate or unknown review: {operation}")
        if review["source_fingerprint"] != operations[operation]:
            raise ValueError(f"source fingerprint differs: {operation}")
        status = review["status"]
        if status not in STATUSES:
            raise ValueError(f"unknown review status: {status}")
        if status in {"supported", "go_mapping"} and review.get("remaining"):
            raise ValueError(f"completed review still has remaining work: {operation}")
        seen.add(operation)
        counts["unresolved"] -= 1
        counts[status] += 1
        contracts += len(review.get("contracts", []))
    return len(operations), counts, len(seen), contracts


def render(total, counts, reviewed, contracts):
    mapped = counts["go_mapping"]
    percent = 100 * mapped / total if total else 0
    return "\n".join([
        START,
        "| 지표 | 현재 값 | 해석 |",
        "|---|---:|---|",
        f"| 고정 소스 전체 선언 | {total:,} | Gophercloud·openstacksdk 선언; inherited/descriptor/Resource 표면은 별도 추적 |",
        f"| 검증된 Go 매핑 | {mapped:,} ({percent:.1f}%) | 전체 연산의 `go_mapping` 판정. 부분 계약 추가만으로 이 수를 늘리지 않음 |",
        f"| source 그대로 지원 | {counts['supported']:,} | `supported` 판정 |",
        f"| 미해결 / 미지원 | {counts['unresolved']:,} / {counts['unsupported']:,} | 미검토 선언도 미해결 집계에 포함 |",
        f"| 연산별 검토 기록 | {reviewed:,} | 아직 개별 기록 없는 선언 {total - reviewed:,} |",
        f"| 기록한 부분·전체 계약 | {contracts:,} | [판정 JSON](../api/sdk_reviews.json)의 계약 항목 수; 테스트 함수 수나 전체 API 완료 수와 다름 |",
        END,
    ])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[3])
    parser.add_argument("--check", action="store_true", help="fail on stale metrics without writing")
    args = parser.parse_args()
    try:
        total, counts, reviewed, contracts = summary(args.root)
        path = args.root / "docs/implementation-plan.md"
        text = path.read_text()
        if text.count(START) != 1 or text.count(END) != 1:
            raise ValueError("progress markers must occur exactly once")
        start, end = text.index(START), text.index(END) + len(END)
        if end < start:
            raise ValueError("progress markers are out of order")
        fresh = text[:start] + render(total, counts, reviewed, contracts) + text[end:]
        if args.check and fresh != text:
            raise ValueError("implementation plan metrics are stale; run make progress")
        if not args.check and fresh != text:
            path.write_text(fresh)
        print(f"Declared API progress: {counts['go_mapping']}/{total} Go mappings; "
              f"supported={counts['supported']} unresolved={counts['unresolved']} "
              f"unsupported={counts['unsupported']}; reviews={reviewed} contracts={contracts}")
        return 0
    except (OSError, ValueError, KeyError, TypeError) as error:
        print(error, file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
