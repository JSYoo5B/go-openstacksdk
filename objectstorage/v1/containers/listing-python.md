# Listing Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--prefix", default="assets/")
    parser.add_argument("--page-size", type=int)
    parser.add_argument("--max-items", type=int, default=0)
    args = parser.parse_args()
    query = {"prefix": args.prefix, "max_items": args.max_items}
    if args.page_size is not None:
        query["limit"] = args.page_size
    conn = openstack.connect()
    for container in conn.object_store.containers(**query):
        print(container.name, container.count, container.bytes)

if __name__ == "__main__":
    main()
```

See `Example_listing` in [example_listing_test.go](example_listing_test.go), compiled by `go test`.

## References

- [Python List Contracts Test](python_list_contracts_test.go)
