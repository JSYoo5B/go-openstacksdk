# Cloud Object Search Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect()
objects = conn.search_objects("assets", name="logs-*", filters={"content_length": 5})
print([obj.name for obj in objects])
print(conn.search_objects("assets", filters="[].name"))
print(conn.search_objects("assets", filters="length(@)"))
```

See `Example_cloudObjectSearch` in [example_cloud_object_search_test.go](example_cloud_object_search_test.go), compiled by `go test`.

## References

- [Cloud Object Search Test](cloud_object_search_test.go)
- [Connection Swift Object Search Test](../connectiontest/connection_swift_object_search_test.go)
