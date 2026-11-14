# Create Volume

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_create_volume_test.go` in [example_create_volume_test.go](example_create_volume_test.go), compiled by `go test`.

```python
volume = conn.create_volume(10, name="data", metadata={"owner": "worker"})
boot_volume = conn.create_volume(20, image="ubuntu", bootable=True)
```

## References

- [..](../README.md)
- [Volume SetBootable API](v3/README.md)
- [BlockStorageCloudMixin.create_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L175-L232)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L494-L503)
- [Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L54-L144)
- [hint request preparation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L446-L464)
- [iterate_timeout](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L102)
- [Attach Volume](attach-volume.md)
- [Detach Volume](detach-volume.md)
