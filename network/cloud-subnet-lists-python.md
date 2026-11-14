# Cloud Subnet Lists Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=["list", "search"], default="list")
parser.add_argument("--name", default="")
parser.add_argument("--filters", default="{}")
args = parser.parse_args()
filters = json.loads(args.filters)
with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "list":
        values = conn.list_subnets(filters=filters)
    else:
        values = conn.search_subnets(args.name, filters=filters)
    print(json.dumps([
        value.to_dict() if hasattr(value, "to_dict") else value
        for value in values
    ], indent=2))
```

See `Example_cloudSubnetLists` in [example_cloud_subnet_lists_test.go](example_cloud_subnet_lists_test.go), compiled by `go test`.

## References

- [Install](../docs/install.md)
- [Subnet](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/subnet.py#L121-L176)
