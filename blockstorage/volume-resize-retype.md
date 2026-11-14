# Volume Resize Retype

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_volumeResizeRetype` in [example_volume_resize_retype_test.go](example_volume_resize_retype_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="devstack")
volume_id = "actual-volume-id"
conn.block_storage.extend_volume(volume_id, 20)
conn.block_storage.retype_volume(volume_id, "actual-type-id", migration_policy="never")
conn.block_storage.complete_volume_extend(volume_id, error=False)
```

## References

- [extend/completion Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1084)
- [retype Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1126)
- [Volume.extend/complete_extend](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L165)
- [Volume.retype](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L317)
- [Resource._get_id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1016)
- [Volume Flags](volume-flags.md)
- [Volume Actions](volume-actions.md)
