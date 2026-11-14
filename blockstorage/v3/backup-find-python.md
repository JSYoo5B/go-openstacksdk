# Backup Find Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="source")
backup = conn.block_storage.find_backup(
    "nightly", ignore_missing=False, details=False
)
print(backup.id, backup.name, backup.created_at)
```

See `Example_backupFind` in [example_backup_find_test.go](example_backup_find_test.go), compiled by `go test`.

## References

- [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [Backup Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py)
