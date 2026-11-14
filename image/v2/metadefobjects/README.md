# Metadefobjects

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_metadefobjects_test.go` in [example_metadefobjects_test.go](example_metadefobjects_test.go), compiled by `go test`.

```python
def manage_objects(conn, namespace, name, replacement_name, clear_namespace=False):
    properties = {"vcpus": {"type": "integer", "title": "Virtual CPUs", "minimum": 1}}
    created = conn.image.create_metadef_object(
        namespace, name=name, description="First line\nSecond line",
        properties=properties, required=["vcpus"])
    fetched = conn.image.get_metadef_object(name, namespace)
    rows = list(conn.image.metadef_objects(namespace))
    updated = conn.image.update_metadef_object(
        name, namespace, name=replacement_name,
        description="First line\nSecond line", properties=properties, required=["vcpus"])
    deleted = conn.image.delete_metadef_object(
        replacement_name, namespace, ignore_missing=False)
    cleared = conn.image.delete_all_metadef_objects(namespace) if clear_namespace else None
    return created, fetched, rows, updated, deleted, cleared
```

## References

- [Metadef Object Namespace Record Lists](../../metadef-object-namespace-record-lists.md)
- [proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1543-L1692)
- [  Init  ](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/__init__.py#L561-L601)
- [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_objects.py#L174-L221)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1874-L1937)
- [repository.list](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/__init__.py#L625-L632)
- [serializer.index](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_objects.py#L430-L435)
- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#list-objects)
- [Contracts Test](contracts_test.go)
- [Core Test](core_test.go)
- [Options Test](options_test.go)
- [Connection Image Metadef Objects Test](../../../connectiontest/connection_image_metadef_objects_test.go)
- [Glance Metadef Objects Test](../../../internal/cmd/sdkgen/glance_metadef_objects_test.go)
