# Volume Backups

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_volume_backups_test.go` in [example_volume_backups_test.go](example_volume_backups_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="my-cloud")
backups = conn.list_volume_backups(
    detailed=True,
    filters={"status": "available", "description": "nightly"},
)
matches = conn.search_volume_backups("nightly*", {"status": "available"})
backup = conn.get_volume_backup("nightly")
filtered_backup = conn.get_volume_backup("nightly", filters={})
names = conn.search_volume_backups(filters="[].name")
```

## References

- [List](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L733)
- [Search](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L890)
- [Get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L668)
- [Proxy.backups](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1631)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L540)
- [v3 Backup descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L26)
- [Requests parser](https://requests.readthedocs.io/en/latest/_modules/requests/utils/#parse_header_links)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)
- [Microversions](../docs/microversions.md)
- [V3](v3/README.md)
