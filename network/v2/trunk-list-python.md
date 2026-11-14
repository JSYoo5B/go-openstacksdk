# Trunk List Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--name", default="")
parser.add_argument("--port", default="")
parser.add_argument("--max-items", type=int, default=0)
parser.add_argument("--single-page", action="store_true")
args = parser.parse_args()

conn = openstack.connect(cloud=args.cloud)
query = {"paginated": not args.single_page, "max_items": args.max_items}
if args.name:
    query["name"] = args.name
if args.port:
    query["port_id"] = args.port
for trunk in conn.network.trunks(**query):
    print(json.dumps(trunk.to_dict(), indent=2))
```

See `Example_trunkList` in [example_trunk_list_test.go](example_trunk_list_test.go), compiled by `go test`.

## References

- [Install](../../docs/install.md)
- [Trunk](https://opendev.org/openstack/neutron/src/tag/27.0.0/neutron/conf/policies/trunk.py)
