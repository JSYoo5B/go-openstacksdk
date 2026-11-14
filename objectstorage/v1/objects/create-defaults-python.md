# Create Defaults Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_createDefaults` in [example_create_defaults_test.go](example_create_defaults_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect()
obj = conn.object_store.create_object("backups", "empty.bin", data=b"")
conn.object_store.create_object("backups", "first.txt", data=b"hello",
                                content_type="text/plain")
```

## References

- [owned create workflow](create.md)
- [Create](create.md)
- [Create Defaults Test](create_defaults_test.go)
- [Native Contracts Test](native_contracts_test.go)
