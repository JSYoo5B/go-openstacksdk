# Python Raw Max Items

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--max-items-json", default="null")
args = parser.parse_args()
cap = json.loads(args.max_items_json)
with openstack.connect(cloud=args.cloud) as conn:
    for row in conn.network.networks(limit=10, max_items=cap):
        print(json.dumps(row.to_dict(), sort_keys=True))
```

See `Example_rawMaxItems` in [example_raw_max_items_test.go](example_raw_max_items_test.go), compiled by `go test`.

## References

- [Install](../../../docs/install.md)
