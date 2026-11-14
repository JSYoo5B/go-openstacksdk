# Cloud Container Search Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect()
print([c.name for c in conn.search_containers(name="logs-*", filters={"count": 1})])
print(conn.search_containers(filters="[].name"))
print(conn.search_containers(filters="length(@)"))
```

See `Example_cloudContainerSearch` in [example_cloud_container_search_test.go](example_cloud_container_search_test.go), compiled by `go test`.

## References

- [Cloud Container Search Test](cloud_container_search_test.go)
