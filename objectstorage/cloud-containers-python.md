# Cloud Containers Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect()
container = conn.create_container("photos", public=True)
if container is not None:
    print(container.name, container.read_ACL)

updated = conn.set_container_access("photos", "private")
observed = conn.set_container_access("photos", "public", refresh=True)
print(conn.get_container_access("photos"))
```

See `Example_cloudContainers` in [example_cloud_containers_test.go](example_cloud_containers_test.go), compiled by `go test`.

## References

- [Cloud Containers Test](cloud_containers_test.go)
- [Cloud Records Test](v1/containers/cloud_records_test.go)
