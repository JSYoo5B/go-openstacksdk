# Metadata Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--volume", default="")
parser.add_argument("--snapshot", default="")
parser.add_argument("--action", choices=["fetch", "get", "set", "delete", "clear", "noop"], default="fetch")
parser.add_argument("--metadata", default='{"sdk-example":"python"}')
parser.add_argument("--key", default="sdk-example")
args = parser.parse_args()
if not args.volume and not args.snapshot:
    parser.error("provide --volume or --snapshot")

conn = openstack.connect()
for kind, identity in [("volume", args.volume), ("snapshot", args.snapshot)]:
    if not identity:
        continue
    proxy = conn.block_storage
    if args.action in ("fetch", "get"):
        record = getattr(proxy, f"{args.action}_{kind}_metadata")(identity)
        print(kind, "metadata", record.metadata)
    elif args.action == "set":
        record = getattr(proxy, f"set_{kind}_metadata")(identity, **json.loads(args.metadata))
        print(kind, "cached request delta", record.metadata)
    else:
        keys = None if args.action == "clear" else ([] if args.action == "noop" else [args.key])
        result = getattr(proxy, f"delete_{kind}_metadata")(identity, keys=keys)
        print(kind, "delete return", result)  # Source returns None.
```

See `Example_metadata` in [example_metadata_test.go](example_metadata_test.go), compiled by `go test`.

## References

- [Cinder API](https://docs.openstack.org/api-ref/block-storage/v3/)
- [Policy](https://docs.openstack.org/cinder/latest/configuration/block-storage/policy.html)
- [Python Metadata Test](volumes/python_metadata_test.go)
- [Python Metadata Test](snapshots/python_metadata_test.go)
