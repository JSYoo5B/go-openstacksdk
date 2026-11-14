# Mutations Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect()
container = conn.object_store.create_container(
    "assets", read_ACL=".r:*", is_content_type_detected=False,
    history_location="", metadata={"owner": "team", "disabled": False},
)
print(container.name, container.metadata)
conn.object_store.delete_container("assets")
conn.object_store.delete_container("assets", ignore_missing=False)
```

See `Example_mutations` in [example_mutations_test.go](example_mutations_test.go), compiled by `go test`.

## References

- [Python Record Test](python_record_test.go)
