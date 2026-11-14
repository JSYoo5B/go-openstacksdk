# Detach Volume

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_detach_volume_test.go` in [example_detach_volume_test.go](example_detach_volume_test.go), compiled by `go test`.

```python
server = conn.compute.get_server("server-id")
volume = conn.block_storage.get_volume("volume-id")
conn.detach_volume(server, volume)  # wait=True; success returns None
```

## References

- [..](../README.md)
- [BlockStorageCloudMixin.detach_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L398-L429)
- [Compute delete_volume_attachment](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L2542-L2578)
- [Cinder waiter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3775-L3815)
- [Resource waiter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591-L2667)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L2502-L2540)
- [cloud get_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L82-L122)
- [Attach Volume](attach-volume.md)
