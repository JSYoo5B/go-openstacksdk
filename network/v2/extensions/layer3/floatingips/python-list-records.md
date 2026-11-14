# Python List Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--dns-name", default="app")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    rows = conn.network.ips(status="ACTIVE", dns_name=args.dns_name)
    print(json.dumps([row.to_dict() for row in rows], indent=2))
```

See `Example_listRecords` in [example_list_records_test.go](example_list_records_test.go), compiled by `go test`.

## References

- [Install](../../../../../docs/install.md)
