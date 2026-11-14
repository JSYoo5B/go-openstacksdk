# Server Find Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--server", required=True)
parser.add_argument("--basic", action="store_true")
parser.add_argument("--all-projects", action="store_true")
parser.add_argument("--strict", action="store_true")
args = parser.parse_args()

conn = openstack.connect(cloud=args.cloud)
server = conn.compute.find_server(
    args.server,
    ignore_missing=not args.strict,
    details=not args.basic,
    all_projects=args.all_projects,
)
print(json.dumps(None if server is None else server.to_dict(), indent=2))
```

See `Example_serverFind` in [example_server_find_test.go](example_server_find_test.go), compiled by `go test`.

## References

- [Install](../../docs/install.md)
