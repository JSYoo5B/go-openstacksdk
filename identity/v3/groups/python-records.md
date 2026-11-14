# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import json
import openstack

def main():
    conn = openstack.connect()
    found = conn.identity.find_group(
        "ops", domain_id="domain-id", skip_cache=True,
        error_message="Group lookup failed",
    )
    if found is not None:
        print(json.dumps(found.to_dict(), indent=2, ensure_ascii=False))

    for row in conn.identity.groups(
        domain_id="domain-id", description="operations", limit=100,
        max_items=100,
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))

    selected = conn.search_groups(
        name_or_id="ops-*", domain_id="domain-id",
        filters={"description": "operations"},
    )
    print(json.dumps([row.to_dict() for row in selected], indent=2))

if __name__ == "__main__":
    main()
```

See `Example_groupsFindRecord` in [example_records1_test.go](example_records1_test.go), compiled by `go test`.

See `Example_groupsProxyListRecords` in [example_records_fragment1_test.go](example_records_fragment1_test.go), compiled by `go test`.

## References

- [Policy](https://docs.openstack.org/keystone/2025.2/configuration/policy.html)
- [Group](https://opendev.org/openstack/keystone/src/tag/28.0.0/keystone/common/policies/group.py)
- [Groups](https://opendev.org/openstack/keystone/src/tag/28.0.0/keystone/api/groups.py)
- [User Find](../users/python-record-find.md)
- [Project Find](../projects/python-record-reads.md)
