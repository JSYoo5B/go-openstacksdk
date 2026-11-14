# Snapshot Find Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev", block_storage_default_microversion="3.65")
snapshot = conn.block_storage.find_snapshot("daily")
summary = conn.block_storage.find_snapshot("daily", details=False)
strict = conn.block_storage.find_snapshot(
    "daily", details=True, all_projects=True, ignore_missing=False)
```

See `Example_snapshotFind` in [example_snapshot_find_test.go](example_snapshot_find_test.go), compiled by `go test`.

## References

- [find_snapshot proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L154-L193)
- [Snapshot descriptor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/snapshot.py)
- [Install](../../docs/install.md)
- [Python Find Contracts Test](snapshots/python_find_contracts_test.go)
