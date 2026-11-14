# Volume Backup Export

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_volume_backup_export_test.go` in [example_volume_backup_export_test.go](example_volume_backup_export_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="devstack", block_storage_api_version="3")
response = conn.export_volume_backup(backup_id)
print(response.status_code, response.headers, response.content)
```

## References

- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L648)
- [deprecated v3 `Proxy.export_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1824)
- [`Backup.export`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L184)
- [`Proxy.export_backup`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1812)
- [`Backup.export_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L173)
- [JSON result](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L281)
- [typed `BackupRecord`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/results.go#L201)
- [Volume Backups](volume-backups.md)
- [Volume Backup Mutations](volume-backup-mutations.md)
- [..](../README.md)
