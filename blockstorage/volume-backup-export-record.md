# Volume Backup Export Record

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_volume_backup_export_record_test.go` in [example_volume_backup_export_record_test.go](example_volume_backup_export_record_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="devstack", block_storage_api_version="3")
value = conn.block_storage.export_backup(backup_id)
print(value)

response = conn.block_storage.export_record(backup_id)
print(response.status_code, response.headers, response.content)
```

## References

- [`Proxy.export_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1824)
- [`Proxy.export_backup`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1812)
- [`Backup.export_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L173)
- [`Backup.export`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L184)
- [Response.json](https://github.com/psf/requests/blob/v2.32.5/src/requests/models.py#L947)
- [compat](https://github.com/psf/requests/blob/v2.32.5/src/requests/compat.py#L57-L64)
- [guess_json_utf](https://github.com/psf/requests/blob/v2.32.5/src/requests/utils.py#L947)
- [`backups.Export`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L281)
- [`BackupRecord`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/results.go#L201)
- [Cloud raw export](volume-backup-export.md)
- [Volume Backups](volume-backups.md)
- [Volume Backup Mutations](volume-backup-mutations.md)
- [..](../README.md)
