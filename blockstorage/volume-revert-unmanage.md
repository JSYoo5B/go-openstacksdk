# Volume Revert Unmanage

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_volumeRevertUnmanage` in [example_volume_revert_unmanage_test.go](example_volume_revert_unmanage_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="devstack")
conn.block_storage.revert_volume_to_snapshot(volume_id, snapshot_id)
conn.block_storage.unmanage_volume(volume_id)
```

## References

- [revert Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1220-L1239)
- [unmanage Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1293-L1303)
- [Volume.revert_to_snapshot](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L226-L232)
- [Volume.unmanage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L311-L315)
- [Keystoneauth5.16.0 version_match](https://github.com/openstack/keystoneauth/blob/e759cf88c072d5bc7e21fc3fa884708ace9630af/keystoneauth1/discover.py#L433-L458)
- [Volume Actions](volume-actions.md)
- [Volume Migration Reset](volume-migration-reset.md)
- [service API](v3/volumes/README.md)
