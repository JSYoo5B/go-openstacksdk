# Attach Volume

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_attach_volume_test.go` in [example_attach_volume_test.go](example_attach_volume_test.go), compiled by `go test`.

```python
server = conn.compute.get_server("server-id")
volume = conn.block_storage.get_volume("volume-id")
attachment = conn.attach_volume(server, volume)  # wait=True, timeout=None
```

## References

- [..](../README.md)
- [Nova VolumeAttachment API](../compute/v2/README.md)
- [BlockStorageCloudMixin.attach_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L431-L492)
- [Cinder wait_for_status](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3775-L3815)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591-L2667)
- [iterate_timeout](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L102)
