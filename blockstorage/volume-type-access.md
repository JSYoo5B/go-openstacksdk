# Volume Type Access

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_volume_type_access_test.go` in [example_volume_type_access_test.go](example_volume_type_access_test.go), compiled by `go test`.

```python
access = conn.get_volume_type_access("fast")
conn.add_volume_type_access("fast", "project-id")
conn.remove_volume_type_access("fast", "project-id")
```

## References

- [Volume Types](volume-types.md)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L949)
- [pinned Type.get_private_access](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/type.py#L119)
- [Type](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/type.py#L136)
- [Exceptions](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/exceptions.py#L214)
- [native pager status](https://github.com/gophercloud/gophercloud/blob/v2.15.0/pagination/http.go#L59)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumetypes/results.go#L159)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumetypes/requests.go#L287)
