# Cloud QoS Policy Mutations Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("create", "update"), default="create")
parser.add_argument("--identity", default="")
parser.add_argument("--attrs", default="{}")
args = parser.parse_args()
if args.mode == "update" and not args.identity:
    parser.error("update identity is required")
attrs = json.loads(args.attrs)
if not isinstance(attrs, dict):
    parser.error("attrs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "create":
        value = conn.create_qos_policy(**attrs)
    else:
        value = conn.update_qos_policy(args.identity, **attrs)
    print(json.dumps(None if value is None else value.to_dict(), indent=2))
```

See `Example_cloudQosPolicyMutations` in [example_cloud_qos_policy_mutations_test.go](example_cloud_qos_policy_mutations_test.go), compiled by `go test`.

## References

- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L90-L134)
- [Base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L28-L46)
- [Attributes](https://github.com/openstack/neutron-lib/blob/3.21.0/neutron_lib/api/attributes.py#L24-L56)
- [QoS](https://github.com/openstack/neutron-lib/blob/3.21.0/neutron_lib/api/definitions/qos.py#L69-L76)
- [Install](../docs/install.md)
- [cloud create/update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1432-L1510)
- [Neutron Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5080-L5210)
- [QoSPolicy descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_policy.py)
- [Resource dirty/create/commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
