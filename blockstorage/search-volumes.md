# Search Volumes

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_search_volumes_test.go` in [example_search_volumes_test.go](example_search_volumes_test.go), compiled by `go test`.

```python
volumes = conn.search_volumes(
    "data*", filters={"metadata": {"owner": "worker"}, "is_bootable": False}
)
ids = conn.search_volumes(filters="[?size >= `10`].id")
volume = conn.get_volume("data")
```
