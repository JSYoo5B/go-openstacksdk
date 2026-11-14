# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import json
import openstack

def main():
    conn = openstack.connect()
    for row in conn.identity.regions(
        parent_region_id="Root", description="operations", limit=100,
        max_items=100,
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_records` in [example_records_test.go](example_records_test.go), compiled by `go test`.

## References

- [Region](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/region.py)
