# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import types
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("list", "find", "proxy"), default="list")
parser.add_argument("--extension", default="qos")
parser.add_argument("--strict", action="store_true")
parser.add_argument("--jmes", default="")
parser.add_argument("--list-base-path")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "find":
        query = {"skip_cache": True, "error_message": "Extension lookup failed"}
        if args.list_base_path is not None:
            query["list_base_path"] = args.list_base_path
        value = conn.network.find_extension(
            args.extension, ignore_missing=not args.strict, **query
        )
        print(json.dumps(None if value is None else value.to_dict(), indent=2))
    else:
        query = {}
        if args.mode == "list":
            query["alias"] = args.extension
        else:
            query["jmespath_filters"] = args.jmes
        result = conn.network.extensions(**query)
        if isinstance(result, types.GeneratorType):
            for value in result:
                print(json.dumps(value.to_dict(), indent=2))
        else:
            print(result)
```

See `Example_records` in [example_records_test.go](example_records_test.go), compiled by `go test`.

## References

- [Extensions](https://github.com/openstack/neutron/blob/27.0.0/neutron/api/extensions.py#L100-L127)
- [Install](../../../docs/install.md)
- [Extension class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/extension.py#L16)
- [find/list proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L1588-L1626)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2489)
- [list element cast](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86-L113)
