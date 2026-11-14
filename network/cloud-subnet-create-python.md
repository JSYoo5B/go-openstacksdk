# Cloud Subnet Create Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--network", required=True)
parser.add_argument("--kwargs", default='{"cidr":"10.61.0.0/24"}')
args = parser.parse_args()
kwargs = json.loads(args.kwargs)
if not isinstance(kwargs, dict):
    parser.error("kwargs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    subnet = conn.create_subnet(args.network, **kwargs)
    print(json.dumps(subnet.to_dict(), indent=2))
```

See `Example_cloudSubnetCreate` in [example_cloud_subnet_create_test.go](example_cloud_subnet_create_test.go), compiled by `go test`.

## References

- [Install](../docs/install.md)
- [Subnet](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/subnet.py#L60-L118)
- [cloud create_subnet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2401-L2569)
- [Subnet descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet.py)
- [Proxy._create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L771-L800)
