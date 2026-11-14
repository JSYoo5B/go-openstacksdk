# Volume Backup Import

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_volumeBackupImport` in [example_volume_backup_import_test.go](example_volume_backup_import_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="devstack")
backup = conn.block_storage.import_backup(
    service="cinder.backup.drivers.swift.SwiftBackupDriver",
    url="opaque-record",
)
print(backup.id, backup.to_dict())
```

## References

- [`Proxy.import_backup`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1802)
- [`Backup.import_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L146)
- [`Resource._get_microversion`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1418)
- [`maximum_supported_microversion`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L358)
- [native Import](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L306)
- [Volume Backups](volume-backups.md)
- [Backup export record](volume-backup-export-record.md)
- [Backup restore/reset](volume-backup-actions.md)
