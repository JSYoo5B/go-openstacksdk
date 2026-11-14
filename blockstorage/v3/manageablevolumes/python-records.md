# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import json
import openstack

with openstack.connect() as conn:
    for row in conn.block_storage.manageable_volumes(details=False, host="host@backend", limit=20, max_items=20):
        print(json.dumps(row.to_dict(), indent=2))
    for row in conn.block_storage.manageable_snapshots(details=True, host="host@backend", max_items=20):
        print(json.dumps(row.to_dict(), indent=2))
```

See `Example_records` in [example_records_test.go](example_records_test.go), compiled by `go test`.
