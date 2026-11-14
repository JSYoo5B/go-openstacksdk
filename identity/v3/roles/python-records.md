# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import json
import openstack

def main():
    conn = openstack.connect()
    for role in conn.identity.roles(description="operations", limit=2,
                                    max_items=2):
        print(json.dumps(role.to_dict(), indent=2, ensure_ascii=False))

    listed = conn.list_roles(description="operations")
    print(json.dumps([role.to_dict() for role in listed], indent=2))
    selected = conn.search_roles("read*", filters={"options": {"immutable": False}})
    print(json.dumps([role.to_dict() for role in selected], indent=2))
    count = conn.search_roles(filters="length(@)")
    print(json.dumps(count, indent=2))

if __name__ == "__main__":
    main()
```

See `Example_records` in [example_records_test.go](example_records_test.go), compiled by `go test`.

## References

- [openstacksdk ef55d7d](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/role.py)
- [Source Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)
- [Keystone28.0.0 Role policy](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/role.py#L99-L119)
- [controller](https://github.com/openstack/keystone/blob/28.0.0/keystone/api/roles.py#L45-L72)
- [Base](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/base.py)
- [Policy](https://docs.openstack.org/keystone/2025.2/configuration/policy.html)
- [Python Record Find](python-record-find.md)
