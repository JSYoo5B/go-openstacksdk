# Consistency Group Snapshots Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("operation", choices=["get", "find", "list", "create", "delete"])
parser.add_argument("--identity")
parser.add_argument("--group")
parser.add_argument("--name")
parser.add_argument("--attrs", default="{}")
parser.add_argument("--query", default="{}")
parser.add_argument("--strict", action="store_true")
parser.add_argument("--summary", action="store_true")
args = parser.parse_args()
conn = openstack.connect()
if args.operation in {"get", "find", "delete"} and args.identity is None:
    parser.error("--identity is required")
if args.operation == "get":
    value = conn.block_storage.get_consistency_group_snapshot(args.identity)
    print(value.to_dict())
elif args.operation == "find":
    value = conn.block_storage.find_consistency_group_snapshot(
        args.identity, ignore_missing=not args.strict, details=not args.summary
    )
    print(None if value is None else value.to_dict())
elif args.operation == "list":
    for value in conn.block_storage.consistency_group_snapshots(
        details=not args.summary, **json.loads(args.query)
    ):
        print(value.to_dict())
elif args.operation == "create":
    attrs = json.loads(args.attrs)
    if args.group is not None:
        attrs["consistencygroup_id"] = args.group
    if args.name is not None:
        attrs["name"] = args.name
    print(conn.block_storage.create_consistency_group_snapshot(**attrs).to_dict())
else:
    conn.block_storage.delete_consistency_group_snapshot(
        args.identity, ignore_missing=not args.strict
    )
    print("delete returned")
```

See `Example_consistencyGroupSnapshots` in [example_consistency_group_snapshots_test.go](example_consistency_group_snapshots_test.go), compiled by `go test`.

```sh
python consistency_snapshots.py list --query '{"limit": 20, "status": "available"}'
go run ./cmd/consistency-snapshots -operation list -query '{"limit": 20, "status": "available"}'
python consistency_snapshots.py create --group source-consistency-group-id --name nightly
go run ./cmd/consistency-snapshots -operation create -group source-consistency-group-id -name nightly
python consistency_snapshots.py find --identity nightly --strict
go run ./cmd/consistency-snapshots -operation find -identity nightly -strict
```

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2503-L2640)
- [Consistency Group Snapshot](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/consistency_group_snapshot.py#L16-L49)
- [Cgsnapshots](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/contrib/cgsnapshots.py)
- [default standard_extensions](https://github.com/openstack/cinder/blob/27.0.0/cinder/common/config.py)
- [API](https://github.com/openstack/cinder/blob/27.0.0/cinder/group/api.py)
- [Group Snapshots](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/group_snapshots.py)
