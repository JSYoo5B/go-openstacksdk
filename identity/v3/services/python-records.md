# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import json
import os
import openstack

def print_value(value):
    if hasattr(value, "to_dict"):
        value = value.to_dict()
    print(json.dumps(value, indent=2, ensure_ascii=False))

def main():
    conn = openstack.connect()
    identifier = os.environ["IDENTITY_SERVICE_ID"]
    for row in conn.identity.services(
        type="compute", is_enabled=True, max_items=5
    ):
        print_value(row)
    print_value(conn.identity.services(jmespath_filters="[*].id"))
    print_value(conn.search_services(name_or_id="n*", filters="length(@)"))
    print_value(conn.get_service(identifier))
    print_value(conn.get_service(identifier, filters={}))

if __name__ == "__main__":
    main()
```

See `Example_records` in [example_records_test.go](example_records_test.go), compiled by `go test`.

## References

- [Service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/service.py)
- [Proxy.services/find_service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1274)
- [Cloud search_services/get_service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L600)
- [Policy](https://docs.openstack.org/keystone/2026.1/configuration/policy.html)
- [Service](https://docs.openstack.org/keystone/2026.1/_modules/keystone/common/policies/service.html)
