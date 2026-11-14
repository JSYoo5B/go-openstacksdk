# Python Manage

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--action", choices=("manage", "unmanage"), required=True)
parser.add_argument("--identity", required=True)
parser.add_argument("--reference", default='{"source-name":"backend-snapshot"}')
args = parser.parse_args()
with openstack.connect() as conn:
    if args.action == "manage":
        row = conn.block_storage.manage_snapshot(volume_id=args.identity, ref=json.loads(args.reference))
        print(json.dumps(row.to_dict(), indent=2))
    else:
        conn.block_storage.unmanage_snapshot(args.identity)
```

See `Example_manage` in [example_manage_test.go](example_manage_test.go), compiled by `go test`.
