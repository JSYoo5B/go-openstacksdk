# Address Group List Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
rows = list(conn.network.address_groups(
    project_id="PROJECT_ID",
    addresses=["10.0.0.0/24"],
    max_items=20,
))
for row in rows:
    print(row.to_dict())
```

See `Example_addressGroupList` in [example_address_group_list_test.go](example_address_group_list_test.go), compiled by `go test`.

## References

- [Install](../../docs/install.md)
- [Address Group](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/address_group.py)
