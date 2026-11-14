# Cloud QoS Rule Updates Python

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
parser.add_argument("--rule", default="")
parser.add_argument("--kwargs", default="{}")
args = parser.parse_args()
kwargs = json.loads(args.kwargs)
if not isinstance(kwargs, dict):
    parser.error("kwargs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    if args.kind == "bandwidth":
        result = conn.update_qos_bandwidth_limit_rule(args.policy, args.rule, **kwargs)
    elif args.kind == "dscp":
        result = conn.update_qos_dscp_marking_rule(args.policy, args.rule, **kwargs)
    else:
        result = conn.update_qos_minimum_bandwidth_rule(args.policy, args.rule, **kwargs)
    print(json.dumps(None if result is None else result.to_dict(), indent=2))
```

See `Example_cloudQosRuleUpdates` in [example_cloud_qos_rule_updates_test.go](example_cloud_qos_rule_updates_test.go), compiled by `go test`.

## References

- [Install](../docs/install.md)
- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L244-L260)
- [DSCP](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L376-L392)
- [Minimum](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L452-L468)
- [Base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L56-L68)
- [BW update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1666-L1731)
- [DSCP update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1877-L1930)
- [Minimum update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2070-L2129)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
