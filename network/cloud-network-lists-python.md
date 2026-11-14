# Cloud Network Lists Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("list", "search"), default="list")
parser.add_argument("--name", default="")
parser.add_argument("--filters", default="{}")
args = parser.parse_args()
filters = json.loads(args.filters)
with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "search":
        result = conn.search_networks(args.name, filters=filters)
    else:
        result = conn.list_networks(filters=filters)
    values = [value.to_dict() if hasattr(value, "to_dict") else value for value in result]
    print(json.dumps(values, indent=2))
```

See `Example_cloudNetworkLists` in [example_cloud_network_lists_test.go](example_cloud_network_lists_test.go), compiled by `go test`.

## References

- [Neutron 27 network policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/network.py)
