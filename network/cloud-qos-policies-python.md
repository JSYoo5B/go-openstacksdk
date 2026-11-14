# Cloud QoS Policies Python

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
parser.add_argument("--filters", default="null")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    filters = json.loads(args.filters)
    if args.mode == "search":
        values = conn.search_qos_policies(name_or_id=args.name, filters=filters)
    else:
        values = conn.list_qos_policies(filters=filters)
    for value in values:
        if hasattr(value, "to_dict"):
            value = value.to_dict()
        print(json.dumps(value, indent=2))
```

See `Example_cloudQosPolicies` in [example_cloud_qos_policies_test.go](example_cloud_qos_policies_test.go), compiled by `go test`.

## References

- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L42-L71)
- [Install](../docs/install.md)
- [Cloud list/search](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L253-L348)
- [extension gate](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L43-L50)
- [Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866-L916)
- [QoSPolicy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_policy.py)
