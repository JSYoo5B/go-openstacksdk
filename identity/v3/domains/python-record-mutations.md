# Python Record Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("create", "update"), required=True)
parser.add_argument("--identity", default="")
parser.add_argument("--name", default="example-domain")
args = parser.parse_args()
if args.mode == "update" and not args.identity:
    parser.error("--identity is required for update")
with openstack.connect(cloud=args.cloud) as conn:
    attrs = {"description": None, "is_enabled": False,
             "options": {"immutable": False}}
    if args.mode == "create":
        row = conn.identity.create_domain(name=args.name, **attrs)
    else:
        row = conn.identity.update_domain(args.identity, **attrs)
        unchanged = conn.identity.update_domain(args.identity)
        print(json.dumps(unchanged.to_dict(), indent=2))
    print(json.dumps(row.to_dict(), indent=2))
```

See `Example_recordMutations` in [example_record_mutations_test.go](example_record_mutations_test.go), compiled by `go test`.

## References

- [Python Records](python-records.md)
- [Install](../../../docs/install.md)
