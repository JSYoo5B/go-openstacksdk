# Cloud Endpoints Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

def jsonable(value):
    if hasattr(value, "to_dict"):
        return value.to_dict()
    if isinstance(value, list):
        return [jsonable(item) for item in value]
    if isinstance(value, dict):
        return {key: jsonable(item) for key, item in value.items()}
    return value

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="system-reader")
    parser.add_argument("--id", default="")
    parser.add_argument("--filters", default="")
    parser.add_argument("--get", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    filters = json.loads(args.filters) if args.filters else None
    if args.get:
        value = conn.get_endpoint(args.id, filters=filters)
    else:
        value = conn.search_endpoints(id=args.id, filters=filters)
    print(json.dumps(jsonable(value), indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_cloudEndpoints` in [example_cloud_endpoints_test.go](example_cloud_endpoints_test.go), compiled by `go test`.

## References

- [`IdentityCloudMixin.search_endpoints/get_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py)
- [`_utils._filter_list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py)
- [Python Records](v3/endpoints/python-records.md)
- [Endpoint](https://github.com/openstack/keystone/blob/27.0.0/keystone/common/policies/endpoint.py)
- [Base](https://github.com/openstack/keystone/blob/27.0.0/keystone/common/policies/base.py)
