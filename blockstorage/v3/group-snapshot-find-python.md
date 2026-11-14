# Group Snapshot Find Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("identity")
parser.add_argument("--strict", action="store_true")
parser.add_argument("--summary", action="store_true")
args = parser.parse_args()
conn = openstack.connect()
value = conn.block_storage.find_group_snapshot(
    args.identity,
    ignore_missing=not args.strict,
    details=not args.summary,
)
print(None if value is None else value.to_dict())
```

See `Example_groupSnapshotFind` in [example_group_snapshot_find_test.go](example_group_snapshot_find_test.go), compiled by `go test`.

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2690-L2719)
- [Policy](https://docs.openstack.org/cinder/latest/configuration/block-storage/policy.html)
