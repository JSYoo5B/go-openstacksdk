# Python Record Find

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--identity", required=True)
parser.add_argument("--domain", required=True)
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    role = conn.identity.find_role(
        args.identity, ignore_missing=False, domain_id=args.domain,
        skip_cache=True, error_message="Role lookup failed",
    )
    print(json.dumps(role.to_dict(), indent=2))
```

See `Example_recordFind` in [example_record_find_test.go](example_record_find_test.go), compiled by `go test`.

## References

- [Python Records](python-records.md)
- [Install](../../../docs/install.md)
