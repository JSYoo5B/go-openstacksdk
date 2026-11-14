# Volume Actions

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_volumeActions` in [example_volume_actions_test.go](example_volume_actions_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="devstack")
volume_id = "actual-volume-id"
conn.block_storage.reserve_volume(volume_id)
conn.block_storage.unreserve_volume(volume_id)
conn.block_storage.begin_volume_detaching(volume_id)
conn.block_storage.abort_volume_detaching(volume_id)
```

## References

- [v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1422)
- [Volume methods](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L405)
- [Source `_get_resource`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563)
- [Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L146)
- [Source utility](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L358)
- [native Reserve/Unreserve/BeginDetaching](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/requests.go#L363)
- [cloud bootable helper](volume-mutations.md)
- [Volume Flags](volume-flags.md)
