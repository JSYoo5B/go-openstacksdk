# Cloud Flavor Ram Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--ram", default="4096", help="minimum RAM as JSON number")
parser.add_argument("--include", default="null", help="name substring as JSON string, or null")
parser.add_argument("--get-extra", action=argparse.BooleanOptionalAction, default=True)
args = parser.parse_args()

with openstack.connect(cloud=args.cloud, compute_api_version="2.55") as conn:
    result = conn.get_flavor_by_ram(
        json.loads(args.ram),
        include=json.loads(args.include),
        get_extra=args.get_extra,
    )
    print(json.dumps(result.to_dict(), indent=2))
```

See `Example_cloudFlavorRam` in [example_cloud_flavor_ram_test.go](example_cloud_flavor_ram_test.go), compiled by `go test`.

## References

- [Install](../docs/install.md)
