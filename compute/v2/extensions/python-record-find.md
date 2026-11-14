# Python Record Find

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--extension", required=True)
parser.add_argument("--strict", action="store_true")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    value = conn.compute.find_extension(args.extension, ignore_missing=not args.strict)
    print(json.dumps(None if value is None else value.to_dict(), indent=2))
```

See `Example_recordFind` in [example_record_find_test.go](example_record_find_test.go), compiled by `go test`.

## References

- [find_extension](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L108)
- [Extension](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/extension.py)
- [Install](../../../docs/install.md)
- [extension policy](https://github.com/openstack/nova/blob/31.0.0/nova/policies/extensions.py)
