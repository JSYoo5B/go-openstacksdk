# Volume Reads

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_volume_reads_test.go` in [example_volume_reads_test.go](example_volume_reads_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="example")
volumes = conn.list_volumes()
volume = conn.get_volume_by_id("volume-id")
exists = conn.volume_exists("data")

volumes = conn.list_volumes(cache=False)
```

## References

- [Volume Id](volume-id.md)
- [Search Volumes](search-volumes.md)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py)
- [v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [v3 Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py)
