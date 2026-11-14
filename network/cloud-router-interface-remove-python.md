# Cloud Router Interface Remove Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--router", required=True)
parser.add_argument("--subnet-json", default="null")
parser.add_argument("--port-json", default="null")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    result = conn.remove_router_interface(
        args.router,
        subnet_id=json.loads(args.subnet_json),
        port_id=json.loads(args.port_json),
    )
    print(json.dumps(result))
```

See `Example_cloudRouterInterfaceRemove` in [example_cloud_router_interface_remove_test.go](example_cloud_router_interface_remove_test.go), compiled by `go test`.

## References

- [Install](../docs/install.md)
- [Router](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/router.py#L384-L398)
