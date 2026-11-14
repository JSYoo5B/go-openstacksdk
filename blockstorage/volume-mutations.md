# Volume Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_volume_mutations_test.go` in [example_volume_mutations_test.go](example_volume_mutations_test.go), compiled by `go test`.

```python
updated = conn.update_volume("data", name="renamed", metadata={})
conn.set_volume_bootable("data")
conn.set_volume_bootable("data", False)
```

## References

- [pinned cloud update_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L234)
- [set_volume_bootable](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L254)
- [Volume Reads](volume-reads.md)
- [pinned _get_volume_kwargs](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L494)
- [pinned Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881)
- [Search Volumes](search-volumes.md)
- [pinned Volume request body](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L446)
- [pinned response translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338)
- [pinned Volume bootable action](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L177)
