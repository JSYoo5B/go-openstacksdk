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
SERVICE_START = "<!-- sdk-service-progress:start -->"
SERVICE_END = "<!-- sdk-service-progress:end -->"
CORE = [
    ("identity", "Identity / Keystone"), ("compute", "Compute / Nova"),
    ("placement", "Placement"), ("network", "Network / Neutron"),
    ("image", "Image / Glance"), ("blockstorage", "Block Storage / Cinder"),
    ("keymanager", "Key Manager / Barbican"), ("objectstorage", "Object Storage / Swift"),
]
LATER = [
    ("loadbalancer", "Load Balancer / Octavia"), ("dns", "DNS / Designate"),
    ("baremetal", "Bare Metal / Ironic"), ("baremetalintrospection", "Bare Metal Introspection"),
    ("db", "Database / Trove"), ("containerinfra", "Container Infra / Magnum"),
    ("container", "Container / Zun"), ("clustering", "Clustering / Senlin"),
    ("orchestration", "Orchestration / Heat"), ("messaging", "Messaging / Zaqar"),
    ("workflow", "Workflow / Mistral"), ("sharedfilesystems", "Shared File System / Manila"),
    ("instanceha", "Instance HA / Masakari"), ("metric", "Metric"),
    ("reservation", "Reservation / Blazar"), ("accelerator", "Accelerator / Cyborg"),
]
SERVICE_ALIASES = {
    "networking": "network", "block_storage": "blockstorage", "key_manager": "keymanager",
    "object_store": "objectstorage", "load_balancer": "loadbalancer", "database": "db",
    "container_infrastructure_management": "containerinfra", "shared_file_system": "sharedfilesystems",
    "instance_ha": "instanceha", "message": "messaging", "baremetal_introspection": "baremetalintrospection",
}
# Group by source responsibility, not every service contacted at runtime.
# NetworkCommon workflows can execute Nova; count them once under Network.
CLOUD_SERVICES = {
    "_accelerator": "accelerator", "_baremetal": "baremetal", "_block_storage": "blockstorage",
    "_coe": "containerinfra", "_compute": "compute", "_dns": "dns", "_identity": "identity",
    "_image": "image", "_network": "network", "_network_common": "network",
    "_object_store": "objectstorage", "_orchestration": "orchestration",
    "_shared_file_system": "sharedfilesystems", "_utils": "objectstorage", "inventory": "compute",
    "openstackcloud": "common",
}


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


def service_owner(operation):
    if operation == "gophercloud:utils.ChooseVersion":
        return "identity"
    source, path = operation.split(":", 1)
    parts = path.split("/")
    owner = parts[0]
    if source == "python" and owner == "cloud":
        owner = CLOUD_SERVICES.get(parts[1])
    elif owner in {"common", "connection"} or owner.startswith("utils."):
        owner = "common"
    owner = SERVICE_ALIASES.get(owner, owner)
    if owner not in {key for key, _ in CORE + LATER} | {"common"}:
        raise ValueError(f"service priority needs classification: {operation}")
    return owner


def service_counts(root):
    catalog = json.loads((root / "api/sdk_support_catalog.json").read_text())
    ledger = json.loads((root / "api/sdk_reviews.json").read_text())
    reviews = {review["operation"]: review for review in ledger["reviews"]}
    counts = {key: Counter() for key, _ in CORE + LATER}
    counts["common"] = Counter()
    for operation in catalog["operations"]:
        count = counts[service_owner(operation)]
        count["total"] += 1
        review = reviews.get(operation)
        status = review["status"] if review else "unreviewed"
        count[status] += 1
    return counts


def completed(count):
    return count["supported"] + count["go_mapping"]


def render_services(counts):
    lines = [SERVICE_START, "| 우선순위 묶음 | 완료 / 전체 | 완료 판정률 | 부분·미해결 검토 | 미검토 | 미지원 |",
             "|---|---:|---:|---:|---:|---:|"]
    groups = [
        ("핵심 서비스 · 1·2단계 합산", CORE),
        ("후속 네트워크 · 3·4단계 합산", LATER[:2]),
        ("후속 베어메탈 · 3·4단계 합산", LATER[2:4]),
        ("후속 나머지 · 3·4단계 합산", LATER[4:]),
        ("서비스 공통 기반", [("common", "공통")]),
        ("전체", CORE + LATER + [("common", "공통")]),
    ]
    for label, owners in groups:
        count = sum((counts[key] for key, _ in owners), Counter())
        done, total = completed(count), count["total"]
        percent = 100 * done / total if total else 0
        lines.append(f"| {label} | {done:,} / {total:,} | {percent:.1f}% | "
                     f"{count['unresolved']:,} | {count['unreviewed']:,} | {count['unsupported']:,} |")
    for label, owners in [("핵심 서비스", CORE), ("후속 서비스 · 네트워크 → 베어메탈 → 나머지", LATER)]:
        lines.extend(["", f"**{label}**", "", "| 서비스 | 완료 / 전체 | 부분·미해결 검토 | 미검토 | 미지원 |",
                      "|---|---:|---:|---:|---:|"])
        for key, name in owners:
            count = counts[key]
            lines.append(f"| {name} | {completed(count):,} / {count['total']:,} | "
                         f"{count['unresolved']:,} | {count['unreviewed']:,} | {count['unsupported']:,} |")
    return "\n".join(lines + [SERVICE_END])


def replace_block(text, start_marker, end_marker, replacement):
    if text.count(start_marker) != 1 or text.count(end_marker) != 1:
        raise ValueError("progress markers must occur exactly once")
    start, end = text.index(start_marker), text.index(end_marker)
    if end < start:
        raise ValueError("progress markers are out of order")
    return text[:start] + replacement + text[end + len(end_marker):]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[3])
    parser.add_argument("--check", action="store_true", help="fail on stale metrics without writing")
    args = parser.parse_args()
    try:
        total, counts, reviewed, contracts = summary(args.root)
        path = args.root / "docs/implementation-plan.md"
        text = path.read_text()
        fresh = replace_block(text, START, END, render(total, counts, reviewed, contracts))
        services = service_counts(args.root)
        fresh = replace_block(fresh, SERVICE_START, SERVICE_END, render_services(services))
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
