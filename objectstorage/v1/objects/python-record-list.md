# Python Record List

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--container", default="assets")
parser.add_argument("--prefix")
parser.add_argument("--mode", choices=("proxy", "cloud"), default="proxy")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "cloud":
        values = conn.list_objects(args.container, prefix=args.prefix)
    else:
        query = {} if args.prefix is None else {"prefix": args.prefix}
        values = conn.object_store.objects(args.container, **query)
    print(json.dumps([value.to_dict() for value in values], indent=2))
```

See `Example_recordList` in [example_record_list_test.go](example_record_list_test.go), compiled by `go test`.

## References

- [`conn.object_store.objects(container, **query)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L249-L271)
- [`conn.list_objects(container, full_listing=True, prefix=None)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L347-L366)
- [Object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L27-L213)
- [Install](../../../docs/install.md)
- [Python Record List Test](python_record_list_test.go)
