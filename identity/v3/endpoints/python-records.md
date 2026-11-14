# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="system-reader")
    parser.add_argument("--service-id", default="")
    parser.add_argument("--region-id", default="")
    args = parser.parse_args()
    query = {"is_enabled": True, "max_items": 100}
    if args.service_id:
        query["service_id"] = args.service_id
    if args.region_id:
        query["region_id"] = args.region_id
    conn = openstack.connect(cloud=args.cloud)
    for row in conn.identity.endpoints(**query):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_records` in [example_records_test.go](example_records_test.go), compiled by `go test`.

## References

- [`Proxy.endpoints`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py)
- [`Endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/endpoint.py)
- [Endpoint](https://github.com/openstack/keystone/blob/27.0.0/keystone/common/policies/endpoint.py)
- [Base](https://github.com/openstack/keystone/blob/27.0.0/keystone/common/policies/base.py)
