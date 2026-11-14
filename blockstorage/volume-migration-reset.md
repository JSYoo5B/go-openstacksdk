# Volume Migration Reset

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_volumeMigrationReset` in [example_volume_migration_reset_test.go](example_volume_migration_reset_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="devstack")
conn.block_storage.reset_volume_status(volume_id, status="available", attach_status="detached")
conn.block_storage.migrate_volume(volume_id, host="host@backend", force_host_copy=True)
conn.block_storage.complete_volume_migration(volume_id, new_volume_id, error=False)
```

## References

- [reset Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1199)
- [migration/completion Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1336)
- [Volume.reset_status](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L209)
- [Volume.migrate/complete_migration](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L330)
- [Keystoneauth5.16.0 version_match](https://github.com/openstack/keystoneauth/blob/e759cf88c072d5bc7e21fc3fa884708ace9630af/keystoneauth1/discover.py#L433-L458)
- [Volume Actions](volume-actions.md)
- [Volume Resize Retype](volume-resize-retype.md)
- [service API](v3/volumes/README.md)
