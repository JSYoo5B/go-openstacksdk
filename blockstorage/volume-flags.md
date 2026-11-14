# Volume Flags

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_volumeFlags` in [example_volume_flags_test.go](example_volume_flags_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="devstack")
volume_id = "actual-volume-id"
conn.block_storage.set_volume_bootable_status(volume_id, False)
conn.block_storage.set_volume_readonly(volume_id)       # default True
conn.block_storage.set_volume_readonly(volume_id, False)
```

## References

- [bootable Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1149)
- [readonly Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1111)
- [Volume methods](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L177)
- [Volume class maximum 3.71](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L146)
- [Volume Actions](volume-actions.md)
- [cloud `SetVolumeBootable`](volume-mutations.md)
- [native `SetBootable`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/requests.go#L644)
