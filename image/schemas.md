# Schemas

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_schemas_test.go` in [example_schemas_test.go](example_schemas_test.go), compiled by `go test`.

```python
def get_all_schemas(conn):
    getters = (
        ("schemas/images", conn.image.get_images_schema),
        ("schemas/image", conn.image.get_image_schema),
        ("schemas/members", conn.image.get_members_schema),
        ("schemas/member", conn.image.get_member_schema),
        ("schemas/tasks", conn.image.get_tasks_schema),
        ("schemas/task", conn.image.get_task_schema),
        ("schemas/metadefs/namespace", conn.image.get_metadef_namespace_schema),
        ("schemas/metadefs/namespaces", conn.image.get_metadef_namespaces_schema),
        ("schemas/metadefs/resource_type", conn.image.get_metadef_resource_type_schema),
        ("schemas/metadefs/resource_types", conn.image.get_metadef_resource_types_schema),
        ("schemas/metadefs/object", conn.image.get_metadef_object_schema),
        ("schemas/metadefs/objects", conn.image.get_metadef_objects_schema),
        ("schemas/metadefs/property", conn.image.get_metadef_property_schema),
        ("schemas/metadefs/properties", conn.image.get_metadef_properties_schema),
        ("schemas/metadefs/tag", conn.image.get_metadef_tag_schema),
        ("schemas/metadefs/tags", conn.image.get_metadef_tags_schema),
    )
    return {path: getter() for path, getter in getters}
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1951-L2161)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#show-image-schema)
- [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/schemas.py#L58-L104)
- [schema serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/schema.py#L82-L240)
- [Schemas Contracts Test](schemas_contracts_test.go)
- [core tests](schemas_core_test.go)
- [option tests](schemas_options_test.go)
- [Connection test](../connectiontest/connection_image_schemas_test.go)
- [generator test](../internal/cmd/sdkgen/glance_schemas_test.go)
