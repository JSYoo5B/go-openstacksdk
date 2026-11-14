# Python Available

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    value = conn.network.find_available_ip()
    print(json.dumps(None if value is None else value.to_dict(), indent=2))
```

See `Example_available` in [example_available_test.go](example_available_test.go), compiled by `go test`.

## References

- [Install](../../../../../docs/install.md)
