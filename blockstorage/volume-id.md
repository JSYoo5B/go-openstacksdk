# Volume Id

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_volume_id_test.go` in [example_volume_id_test.go](example_volume_id_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="example")
volume_id = conn.get_volume_id("data")
if volume_id is None:
    print("no ID returned")
else:
    print(volume_id)
```

## References

- [Volume Reads](volume-reads.md)
- [Search Volumes](search-volumes.md)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L362-L371)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L82-L122)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
