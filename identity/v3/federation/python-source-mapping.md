# Python Source Mapping

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("create", "find", "update"), required=True)
parser.add_argument("--id", default="example-mapping")
args = parser.parse_args()
rules = [{"local": [{"user": {"name": "{0}", "type": "ephemeral"}}],
          "remote": [{"type": "REMOTE_USER", "any_one_of": ["alice"]}]}]
with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "create":
        row = conn.identity.create_mapping(id=args.id, rules=rules,
                                           schema_version="2.0")
    elif args.mode == "update":
        row = conn.identity.update_mapping(args.id, schema_version="2.0")
    else:
        row = conn.identity.find_mapping(args.id)
    print(json.dumps(None if row is None else row.to_dict(), indent=2))
```

See `Example_sourceMapping` in [example_source_mapping_test.go](example_source_mapping_test.go), compiled by `go test`.

## References

- [Install](../../../docs/install.md)
