# Volume Types

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_volume_types_test.go` in [example_volume_types_test.go](example_volume_types_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="example")
types = conn.list_volume_types()
matches = conn.search_volume_types("fast*", filters={"is_public": True})
selected = conn.get_volume_type("fast")
names = conn.search_volume_types(filters="[].name")
```

## References

- [Requests Response.links](https://requests.readthedocs.io/en/latest/_modules/requests/models/#Response.links)
- [parse_header_links](https://requests.readthedocs.io/en/latest/_modules/requests/utils/#parse_header_links)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L66-L79)
- [search](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L916-L947)
- [get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L133-L173)
- [v3 Type descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/type.py)
- [v3 find query](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L474-L496)
