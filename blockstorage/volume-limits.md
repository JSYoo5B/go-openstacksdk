# Volume Limits

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_volume_limits_test.go` in [example_volume_limits_test.go](example_volume_limits_test.go), compiled by `go test`.

## References

- [pinned cloud get_volume_limits](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L348)
- [pinned Cinder Proxy.get_limits](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1918)
- [Resource paging](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2345)
- [Utils](https://requests.readthedocs.io/en/latest/_modules/requests/utils/#parse_header_links)
- [Models](https://requests.readthedocs.io/en/latest/_modules/requests/models/#RequestEncodingMixin._encode_params)
- [Urllib.Parse](https://docs.python.org/3/library/urllib.parse.html#urllib.parse.urlencode)
- [Resource._get_microversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1418)
- [Cinder v3 limits singleton](v3/limits/README.md)
- [pinned Limits hierarchy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/limits.py)
- [pinned field conversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86)
- [Resource constructor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L538)
