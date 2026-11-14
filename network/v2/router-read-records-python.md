# Router Read Records Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--router", default="edge")
parser.add_argument("--mode", choices=("find", "list"), default="list")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "find":
        value = conn.network.find_router(args.router, ignore_missing=False)
        print(json.dumps(value.to_dict(), indent=2))
    else:
        for value in conn.network.routers(name=args.router, max_items=100):
            print(json.dumps(value.to_dict(), indent=2))
```

See `Example_routerReadRecords` in [example_router_read_records_test.go](example_router_read_records_test.go), compiled by `go test`.

## References

- [Install](../../docs/install.md)
- [Router](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/router.py#L177)
