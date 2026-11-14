# Python Records

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
    values = conn.network.vpn_services(status="ACTIVE")
    for value in values:
        print(json.dumps(value.to_dict(), indent=2))
```

See `Example_records` in [example_records_test.go](example_records_test.go), compiled by `go test`.

## References

- [Install](../../../../../docs/install.md)
