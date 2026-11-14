# Cloud Router Interfaces Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import os
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
parser.add_argument("--router", required=True, help='JSON dictionary, e.g. {"id":"router-id"}')
parser.add_argument("--interface-type", default="null", help='JSON None/falsey or "internal"/"external"')
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    ports = conn.list_router_interfaces(json.loads(args.router),
                                        json.loads(args.interface_type))
    print(json.dumps([value.to_dict() for value in ports], ensure_ascii=False, indent=2))
```

See `Example_cloudRouterInterfaces` in [example_cloud_router_interfaces_test.go](example_cloud_router_interfaces_test.go), compiled by `go test`.

## References

- [list_router_interfaces Source](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2205)
- [Port](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/port.py)
- [Cloud Router Interfaces Test](cloud_router_interfaces_test.go)
- [Connection Router Interface Records Test](../connectiontest/connection_router_interface_records_test.go)
