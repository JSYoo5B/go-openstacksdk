# Root Access

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_root_access_test.go` in [example_root_access_test.go](example_root_access_test.go), compiled by `go test`.

```python
from openstack.database.v1.instance import Instance

instance = Instance(id=instance_id)
enabled = instance.is_root_enabled(conn.database)
```

## References

- [`Instance.is_root_enabled(session)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/database/v1/instance.py#L73-L85)
- [Trove Root Enabled Contracts Test](../../../api/trove_root_enabled_contracts_test.go)
- [..](../README.md)
