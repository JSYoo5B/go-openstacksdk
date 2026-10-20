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
    # Unsupported operations, such as routes of retired server APIs, are
    # excluded from the implementation target.
    target = total - counts["unsupported"]
    percent = 100 * mapped / target if target else 0
    return "\n".join([
        START,
        "| Metric | Current value | Interpretation |",
        "|---|---:|---|",
        f"| Pinned source declarations | {total:,} | Gophercloud/openstacksdk declarations; inherited, descriptor and Resource surfaces tracked separately |",
        f"| Implementation target | {target:,} | All declarations excluding unsupported routes, such as retired server APIs |",
        f"| API mappings | {mapped:,} ({percent:.1f}%) | go_mapping decisions against the target; partial verification does not increase this count |",
        f"| Source-equivalent support | {counts['supported']:,} | supported decisions |",
        f"| Unresolved / excluded (unsupported) | {counts['unresolved']:,} / {counts['unsupported']:,} | Unreviewed declarations are included in unresolved totals |",
        f"| Operation reviews | {reviewed:,} | Declarations without individual reviews {total - reviewed:,} |",
        f"| Behavior contracts | {contracts:,} | contracts entries in the [review JSON](../api/sdk_reviews.json); differing granularity prevents using this as an API completion rate |",
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
    lines = [SERVICE_START, "| Priority group | API mappings / target | Mapping rate | Partial/unresolved reviews | Unreviewed | Excluded (unsupported) |",
             "|---|---:|---:|---:|---:|---:|"]
    groups = [
        ("Core services: stages 1 and 2", CORE),
        ("Following network: stages 3 and 4", LATER[:2]),
        ("Following bare metal: stages 3 and 4", LATER[2:4]),
        ("Other following services: stages 3 and 4", LATER[4:]),
        ("Shared foundation", [("common", "Common")]),
        ("Total", CORE + LATER + [("common", "Common")]),
    ]
    for label, owners in groups:
        count = sum((counts[key] for key, _ in owners), Counter())
        done, total = completed(count), count["total"] - count["unsupported"]
        percent = 100 * done / total if total else 0
        lines.append(f"| {label} | {done:,} / {total:,} | {percent:.1f}% | "
                     f"{count['unresolved']:,} | {count['unreviewed']:,} | {count['unsupported']:,} |")
    for label, owners in [("Core services", CORE), ("Following services: network, bare metal, others", LATER)]:
        lines.extend(["", f"**{label}**", "", "| Service | API mappings / target | Partial/unresolved reviews | Unreviewed | Excluded (unsupported) |",
                      "|---|---:|---:|---:|---:|"])
        for key, name in owners:
            count = counts[key]
            lines.append(f"| {name} | {completed(count):,} / {count['total'] - count['unsupported']:,} | "
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
