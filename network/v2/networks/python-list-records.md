# Python List Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import types
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--name")
parser.add_argument("--max-items", type=int, default=0)
parser.add_argument("--jmes")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    query = {}
    if args.name is not None:
        query["name"] = args.name
    if args.max_items:
        query["max_items"] = args.max_items
    if args.jmes is not None:
        query["jmespath_filters"] = args.jmes
    result = conn.network.networks(**query)
    if isinstance(result, types.GeneratorType):
        for value in result:
            print(json.dumps(value.to_dict(), indent=2))
    else:
        print(result)
```

See `Example_listRecords` in [example_list_records_test.go](example_list_records_test.go), compiled by `go test`.

## References

- [Install](../../../docs/install.md)
- [Network](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/network.py#L196-L265)
- [Proxy.networks](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3258)
- [Network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/network.py#L17)
- [Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)
- [Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155)
