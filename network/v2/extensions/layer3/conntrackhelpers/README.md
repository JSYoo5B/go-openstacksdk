# Conntrackhelpers

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
helper = conn.network.create_conntrack_helper(
    router, helper="ftp", protocol="tcp", port=21)
conn.network.update_conntrack_helper(helper, router, port=2121)
for item in conn.network.conntrack_helpers(router, helper="ftp"):
    print(item.id, item.port)
conn.network.delete_conntrack_helper(helper, router)
```

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.
