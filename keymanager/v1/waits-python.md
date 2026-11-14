# Waits Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import openstack
from openstack.key_manager.v1.container import Container
from openstack.key_manager.v1.order import Order
from openstack.key_manager.v1.secret import Secret

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--kind", choices=["container", "order", "secret"], default="order")
parser.add_argument("--id", required=True)
parser.add_argument("--operation", choices=["status", "delete"], default="status")
parser.add_argument("--status", default="ACTIVE")
parser.add_argument("--null-status", action="store_true")
parser.add_argument("--interval", type=float, default=2)
parser.add_argument("--timeout", type=float)
parser.add_argument("--disable-failures", action="store_true")
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
resource_type = {"container": Container, "order": Order, "secret": Secret}[args.kind]
res = resource_type.existing(id=args.id)
options = {"interval": args.interval, "callback": lambda progress: print("progress", progress)}
if args.timeout is not None:
    options["wait"] = args.timeout
if args.operation == "delete":
    result = conn.key_manager.wait_for_delete(res, **options)
else:
    if args.disable_failures:
        options["failures"] = []
    status = None if args.null_status else args.status
    result = conn.key_manager.wait_for_status(res, status, **options)
print(result.to_dict())
```

See `Example_waits` in [example_waits_test.go](example_waits_test.go), compiled by `go test`.

## References

- [Install](../../docs/install.md)
- [Policy](https://docs.openstack.org/barbican/2025.2/configuration/policy.html)
- [ProjectQuota](waits/python-project-quota.md)
- [proxy wait declarations](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L615-L674)
- [shared status/deletion wait](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2585-L2715)
- [Secret.fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/secret.py#L91-L138)
- [timeout iterator](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L100)
