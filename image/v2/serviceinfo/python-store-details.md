# Python Store Details

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

def main():
    conn = openstack.connect()
    for store in conn.image.stores(details=True, is_default=True, paginated=False):
        print(store.id, store.properties)

if __name__ == "__main__":
    main()
```

See `Example_storeDetails` in [example_store_details_test.go](example_store_details_test.go), compiled by `go test`.
