# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import json
import os
import openstack

def main():
    conn = openstack.connect()
    domain = os.environ["USER_QUERY_DOMAIN_ID"]
    for row in conn.identity.users(
        domain_id=domain, is_enabled=True, max_items=10
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))
    listed = conn.list_users(domain_id=domain)
    print(json.dumps([row.to_dict() for row in listed], indent=2))
    searched = conn.search_users(
        name_or_id="team*", domain_id=domain, filters="length(@)"
    )
    print(json.dumps(searched, indent=2))

if __name__ == "__main__":
    main()
```

See `Example_records` in [example_records_test.go](example_records_test.go), compiled by `go test`.

## References

- [User](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/user.py)
- [Proxy.users](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1429)
- [Cloud list_users/search_users](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L290)
- [Policy](https://docs.openstack.org/keystone/2026.1/configuration/policy)
- [Python Record Find](python-record-find.md)
