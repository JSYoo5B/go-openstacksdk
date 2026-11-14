# Cloud Subnet Update Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--identity", required=True)
parser.add_argument("--attrs", default="{}")
args = parser.parse_args()
attrs = json.loads(args.attrs)
if not isinstance(attrs, dict):
    parser.error("attrs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    result = conn.update_subnet(args.identity, **attrs)
    print(json.dumps(None if result is None else result.to_dict(), indent=2))
```

See `Example_cloudSubnetUpdate` in [example_cloud_subnet_update_test.go](example_cloud_subnet_update_test.go), compiled by `go test`.

## References

- [Install](../docs/install.md)
