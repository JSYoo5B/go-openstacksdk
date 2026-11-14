# Cloud QoS Rule Creates Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--kind", choices=("bandwidth", "dscp", "minimum"), default="bandwidth")
parser.add_argument("--policy", required=True)
parser.add_argument("--value", default="0")
parser.add_argument("--kwargs", default="{}")
args = parser.parse_args()
value = json.loads(args.value)
kwargs = json.loads(args.kwargs)
if not isinstance(kwargs, dict):
    parser.error("kwargs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    if args.kind == "bandwidth":
        result = conn.create_qos_bandwidth_limit_rule(args.policy, value, **kwargs)
    elif args.kind == "dscp":
        result = conn.create_qos_dscp_marking_rule(args.policy, value, **kwargs)
    else:
        result = conn.create_qos_minimum_bandwidth_rule(args.policy, value, **kwargs)
    print(json.dumps(result.to_dict(), indent=2))
```

See `Example_cloudQosRuleCreates` in [example_cloud_qos_rule_creates_test.go](example_cloud_qos_rule_creates_test.go), compiled by `go test`.

## References

- [Install](../docs/install.md)
- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L226-L245)
- [DSCP](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L361-L380)
- [Minimum](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L438-L457)
- [Base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L56-L68)
- [BW cloud create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1620-L1668)
- [DSCP cloud create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1844-L1879)
- [Minimum cloud create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2040-L2077)
- [Source Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
