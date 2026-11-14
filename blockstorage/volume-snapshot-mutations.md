# Volume Snapshot Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_volume_snapshot_mutations_test.go` in [example_volume_snapshot_mutations_test.go](example_volume_snapshot_mutations_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="configured-cloud")
snapshot = conn.create_volume_snapshot(
    "volume-id", name="nightly", description="nightly volume snapshot"
)  # force=False, wait=True, timeout=None
accepted = conn.create_volume_snapshot("volume-id", display_name="nightly", wait=False)
deleted = conn.delete_volume_snapshot("nightly")  # wait=False
deleted_after_wait = conn.delete_volume_snapshot("nightly", wait=True, timeout=600)
absent_input = conn.delete_volume_snapshot()  # False, no lookup
```

## References

- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L505)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L799)
- [pinned v3 Snapshot](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/snapshot.py)
- [Volume Snapshots](volume-snapshots.md)
