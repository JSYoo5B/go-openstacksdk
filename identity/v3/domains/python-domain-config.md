# Python Domain Config

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--domain", required=True)
parser.add_argument("--action", choices=("create", "get", "update", "delete"), required=True)
parser.add_argument("--attrs", default='{"identity":{"driver":"ldap"},"ldap":{"url":"ldap://localhost"}}')
parser.add_argument("--strict-delete", action="store_true")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    if args.action == "create":
        value = conn.identity.create_domain_config(args.domain, **json.loads(args.attrs))
    elif args.action == "get":
        value = conn.identity.get_domain_config(args.domain)
    elif args.action == "update":
        value = conn.identity.update_domain_config(args.domain, **json.loads(args.attrs))
    else:
        value = conn.identity.delete_domain_config(args.domain, ignore_missing=not args.strict_delete)
    print(json.dumps(None if value is None else value.to_dict(), indent=2))
```

See `Example_domainConfig` in [example_domain_config_test.go](example_domain_config_test.go), compiled by `go test`.

## References

- [Domain scalar-ID update](python-record-mutations.md)
- [Install](../../../docs/install.md)
