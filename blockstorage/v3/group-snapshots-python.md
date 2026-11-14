# Group Snapshots Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--action", choices=["get", "list", "create", "delete"], default="get")
parser.add_argument("--id", default="group-snapshot-id")
parser.add_argument("--group", default="group-id")
args = parser.parse_args()
conn = openstack.connect()
if args.action == "get":
    print(conn.block_storage.get_group_snapshot(args.id).to_dict())
elif args.action == "list":
    for item in conn.block_storage.group_snapshots(details=False, status="available", limit=10):
        print(item.to_dict())
elif args.action == "create":
    print(conn.block_storage.create_group_snapshot(group_id=args.group, name="sdk-example").to_dict())
else:
    print(conn.block_storage.delete_group_snapshot(args.id, ignore_missing=False))
```

See `Example_groupSnapshots` in [example_group_snapshots_test.go](example_group_snapshots_test.go), compiled by `go test`.

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group_snapshot.py)
- [Policy](https://docs.openstack.org/cinder/latest/configuration/block-storage/policy.html)
- [V3](https://docs.openstack.org/api-ref/block-storage/v3/#group-snapshots-group-snapshots)
- [Python Records Test](groupsnapshots/python_records_test.go)
