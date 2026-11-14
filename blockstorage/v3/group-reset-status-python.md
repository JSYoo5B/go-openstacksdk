# Group Reset Status Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
conn.block_storage.reset_group_status(group_id, "available")
conn.block_storage.reset_group_snapshot_status(snapshot_id, "error")
```

```go
if _, err := groups.New(volumeClient).ResetStatusRecord(ctx, groupID, "available"); err != nil { return err }
if _, err := groupsnapshots.New(volumeClient).ResetStatusRecord(ctx, snapshotID, "error"); err != nil { return err }
```

## References

- [`_proxy.py` reset_group_status](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2250-L2272)
- [`reset_group_snapshot_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2756-L2786)
- [`group.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py)
- [`group_snapshot.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group_snapshot.py)
