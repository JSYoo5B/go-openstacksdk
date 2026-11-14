# Python Server Migrations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--server", required=True)
parser.add_argument("--migration", default="")
parser.add_argument("--action", choices=("list", "get", "abort", "force", "crash"), required=True)
parser.add_argument("--microversion", default="2.80")
parser.add_argument("--strict-abort", action="store_true")
args = parser.parse_args()
if args.action in ("get", "abort", "force") and not args.migration:
    parser.error("--migration is required")
with openstack.connect(cloud=args.cloud, compute_api_version=args.microversion) as conn:
    if args.action == "list":
        for row in conn.compute.server_migrations(args.server):
            print(json.dumps(row.to_dict(), indent=2))
    elif args.action == "get":
        row = conn.compute.get_server_migration(args.migration, args.server)
        print(json.dumps(row.to_dict(), indent=2))
    elif args.action == "abort":
        conn.compute.abort_server_migration(args.migration, args.server,
                                            ignore_missing=not args.strict_abort)
    elif args.action == "force":
        conn.compute.force_complete_server_migration(args.migration, args.server)
    else:
        conn.compute.trigger_server_crash_dump(args.server)
```

See `Example_serverMigrations` in [example_server_migrations_test.go](example_server_migrations_test.go), compiled by `go test`.

## References

- [Install](../../../docs/install.md)
