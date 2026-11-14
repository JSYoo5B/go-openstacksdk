# Volume Backup Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_volume_backup_mutations_test.go` in [example_volume_backup_mutations_test.go](example_volume_backup_mutations_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="my-cloud")
created = conn.create_volume_backup(volume_id)
queued = conn.create_volume_backup(
    volume_id, name="", description="", force=False, wait=False,
    incremental=True, snapshot_id=snapshot_id,
)
deleted = conn.delete_volume_backup("nightly")
forced = conn.delete_volume_backup("nightly", force=True, wait=True, timeout=600)
empty = conn.delete_volume_backup()
```

## References

- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L600)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L758)
- [Backup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L116)
- [V3](v3/README.md)
- [v3 Backup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L26)
- [Volume Backups](volume-backups.md)
- [Backup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L132)
- [native Backup Create](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L60)
- [ForceDelete](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L357)
- [Microversions](../docs/microversions.md)
