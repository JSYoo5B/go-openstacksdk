# Python Quota Class

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--action", choices=("get", "update"), required=True)
parser.add_argument("--name", default="default")
parser.add_argument("--attrs", default='{"volumes":0}')
parser.add_argument("--microversion", default="")
args = parser.parse_args()
config = {"cloud": args.cloud}
if args.microversion:
    config["block_storage_api_version"] = args.microversion
with openstack.connect(**config) as conn:
    if args.action == "get":
        row = conn.block_storage.get_quota_class_set(args.name)
    else:
        row = conn.block_storage.update_quota_class_set(args.name, **json.loads(args.attrs))
    print(json.dumps(row.to_dict(), indent=2))
```

See `Example_quotaClass` in [example_quota_class_test.go](example_quota_class_test.go), compiled by `go test`.

## References

- [Install](../../../docs/install.md)
