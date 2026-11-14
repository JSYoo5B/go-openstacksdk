# Cloud Server Group Reads Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--operation", choices=("search", "get"), default="search")
parser.add_argument("--name")
parser.add_argument("--filters", help="JSON dictionary, string expression or other Source value")
args = parser.parse_args()
kwargs = {}
if args.filters is not None:
    kwargs["filters"] = json.loads(args.filters)

def view(value):
    if hasattr(value, "to_dict"):
        return value.to_dict()
    if isinstance(value, list):
        return [view(item) for item in value]
    return value

with openstack.connect(cloud=args.cloud) as conn:
    if args.operation == "search":
        result = conn.search_server_groups(args.name, **kwargs)
    else:
        result = conn.get_server_group(args.name, **kwargs)
    print(json.dumps(view(result), indent=2))
```

See `Example_cloudServerGroupReads` in [example_cloud_server_group_reads_test.go](example_cloud_server_group_reads_test.go), compiled by `go test`.

## References

- [Install](../docs/install.md)
- [Server Groups](https://github.com/openstack/nova/blob/32.0.0/nova/policies/server_groups.py#L49-L85)
- [search](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L216-L242)
- [get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L728-L772)
- [filter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L43-L145)
- [ServerGroup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_group.py)
- [Resource.find/equality](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
