# Cloud QoS Policy Actions Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("get", "delete"), default="get")
parser.add_argument("--identity", required=True)
parser.add_argument("--filters", default='{"skip_cache":true,"error_message":"QoS policy lookup failed"}')
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "delete":
        print(json.dumps(conn.delete_qos_policy(args.identity)))
    else:
        value = conn.get_qos_policy(args.identity, filters=json.loads(args.filters))
        print(json.dumps(None if value is None else value.to_dict(), indent=2))
```

See `Example_cloudQosPolicyActions` in [example_cloud_qos_policy_actions_test.go](example_cloud_qos_policy_actions_test.go), compiled by `go test`.

## References

- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L42-L165)
- [Install](../docs/install.md)
- [Cloud get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L220-L250)
- [Cloud delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1512-L1536)
- [Resource.find/fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [Proxy delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L670-L713)
